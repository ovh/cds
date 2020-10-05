package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

type ReadWrite struct {
	Store     cache.ScoredSetStore
	ItemID    string
	PrefixKey string
	UsageKey  string
}

type Line struct {
	Number int64  `json:"number"`
	Value  string `json:"value"`
}

func (l Line) Format(f sdk.CDNReaderFormat) ([]byte, error) {
	switch f {
	case sdk.CDNReaderFormatJSON:
		bs, err := json.Marshal(l)
		return bs, sdk.WithStack(err)
	case sdk.CDNReaderFormatText:
		return []byte(l.Value), nil
	}
	return nil, sdk.WithStack(fmt.Errorf("invalid given reader format '%s'", f))
}

func (rw *ReadWrite) get(from uint, to uint) ([]Line, error) {
	res, err := rw.Store.ScoredSetScanWithScores(context.Background(), cache.Key(rw.PrefixKey, rw.ItemID), float64(from), float64(to))
	if err != nil {
		return nil, err
	}
	ls := make([]Line, len(res))
	for i := range res {
		ls[i].Number = int64(res[i].Score)
		var value string
		if err := json.Unmarshal(res[i].Value, &value); err != nil {
			return nil, sdk.WrapError(err, "cannot unmarshal line value from store")
		}
		ls[i].Value = strings.TrimFunc(value, unicode.IsNumber)
		ls[i].Value = strings.TrimPrefix(ls[i].Value, "#")
	}
	return ls, nil
}

func (rw *ReadWrite) maxScore() (float64, error) {
	res, err := rw.Store.ScoredSetScanMaxScore(context.Background(), cache.Key(rw.PrefixKey, rw.ItemID))
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, nil
	}
	return res.Score, nil
}

func (rw *ReadWrite) UpdateUsage() error {
	return rw.Store.ScoredSetAdd(context.Background(), rw.UsageKey, rw.ItemID, float64(time.Now().UnixNano()))
}

// Add new item in cache + update last usage
func (rw *ReadWrite) add(score uint, value string) error {
	itemKey := cache.Key(rw.PrefixKey, rw.ItemID)
	value = strconv.Itoa(int(score)) + "#" + value
	if err := rw.Store.ScoredSetAdd(context.Background(), itemKey, value, float64(score)); err != nil {
		return err
	}
	return nil
}

func (rw *ReadWrite) Close() error {
	if rw.UsageKey == "" {
		return nil
	}
	// Update last usage
	return rw.UpdateUsage()
}
