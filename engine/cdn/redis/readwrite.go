package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	NumberString string `json:"-"`
	Number       int64  `json:"number"`
	Value        string `json:"value"`
}

func (l Line) Format(f ReaderFormat) ([]byte, error) {
	switch f {
	case ReaderFormatJSON:
		var err error
		l.Number, err = strconv.ParseInt(l.NumberString, 10, 64)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot parse line number with value %s", l.NumberString)
		}
		bs, err := json.Marshal(l)
		return bs, sdk.WithStack(err)
	case ReaderFormatText:
		return []byte(l.Value), nil
	}
	return nil, sdk.WithStack(fmt.Errorf("invalid given reader format '%s'", f))
}

func (rw *ReadWrite) get(from uint, to uint) ([]Line, error) {
	res := make([]string, to-from+1)
	if err := rw.Store.ScoredSetScan(context.Background(), cache.Key(rw.PrefixKey, rw.ItemID), float64(from), float64(to), &res); err != nil {
		return nil, err
	}
	ls := make([]Line, len(res))
	for i := range res {
		tmp := strings.SplitN(res[i], "#", 2)
		if len(tmp) != 2 {
			return nil, sdk.WithStack(fmt.Errorf("cannot split line from redis set, length %d != 2 line start with %s...", len(tmp), sdk.StringFirstN(res[i], 10)))
		}
		ls[i].NumberString = tmp[0]
		ls[i].Value = tmp[1]
	}
	return ls, nil
}

func (rw *ReadWrite) card() (int, error) {
	itemKey := cache.Key(rw.PrefixKey, rw.ItemID)
	return rw.Store.SetCard(itemKey)
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
