package redis

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ovh/cds/engine/cache"
)

type ReadWrite struct {
	Store     cache.ScoredSetStore
	ItemID    string
	PrefixKey string
	UsageKey  string
}

func (rw *ReadWrite) get(from uint, to uint) ([]string, error) {
	var res = make([]string, to-from+1)
	if err := rw.Store.ScoredSetScan(context.Background(), cache.Key(rw.PrefixKey, rw.ItemID), float64(from), float64(to), &res); err != nil {
		return res, err
	}
	for i := range res {
		res[i] = strings.TrimFunc(res[i], unicode.IsNumber)
		res[i] = strings.TrimPrefix(res[i], "#")
	}
	return res, nil
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
