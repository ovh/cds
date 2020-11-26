package lru

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	redisLruItemCacheKey = cache.Key("cdn:lru:item")
	redisLruKeyCacheKey  = cache.Key("cdn:lru:key")
)

type Redis struct {
	maxSize int64
	db      *gorp.DbMap
	store   cache.Store
}

// NewRedisLRU instanciates a new Redis LRU
func NewRedisLRU(db *gorp.DbMap, maxSize int64, host string, password string) (*Redis, error) {
	c, err := cache.New(host, password, -1)
	if err != nil {
		return nil, err
	}
	return &Redis{db: db, maxSize: maxSize, store: c}, nil
}

// Exist returns true is the item ID exists
func (r *Redis) Exist(itemID string) (bool, error) {
	itemKey := cache.Key(redisLruItemCacheKey, itemID)
	return r.store.Exist(itemKey)
}

// Remove remove an itemID
func (r *Redis) Remove(itemIDs []string) error {
	for _, itemID := range itemIDs {
		// Delete item
		itemKey := cache.Key(redisLruItemCacheKey, itemID)
		if err := r.store.Delete(itemKey); err != nil {
			return err
		}
		// Delete usage
		btes, _ := json.Marshal(itemID)
		if err := r.store.ScoredSetRem(context.Background(), redisLruKeyCacheKey, string(btes)); err != nil {
			return sdk.WithStack(err)
		}
	}
	return nil
}

// RemoveOldest removes the oldest entry
func (r *Redis) RemoveOldest() error {
	var keys []string
	if err := r.store.ScoredSetRange(context.Background(), redisLruKeyCacheKey, 0, 0, &keys); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return sdk.WithStack(r.Remove(keys))
}

// Len returns the number of elements in that cache
func (r *Redis) Len() (int, error) {
	return r.store.SetCard(redisLruKeyCacheKey)
}

func (s *Redis) Card(itemID string) (int, error) {
	itemKey := cache.Key(redisLruItemCacheKey, itemID)
	return s.store.SetCard(itemKey)
}

// Size of the cache
func (r *Redis) Size() (int64, error) {
	lenght, err := r.Len()
	if err != nil {
		return 0, err
	}
	if lenght == 0 {
		return 0, nil
	}

	var itemIDs []string
	if err := r.store.ScoredSetRange(context.Background(), redisLruKeyCacheKey, 0, -1, &itemIDs); err != nil {
		return 0, err
	}

	// DB Request
	return item.ComputeSizeByIDs(r.db, itemIDs)
}

// MaxSize returns the maxSize of the cache
func (r *Redis) MaxSize() int64 {
	return r.maxSize
}

// Clear clears the cache, removing old item keys
func (r *Redis) Clear() error {
	itemKeys := cache.Key(redisLruItemCacheKey, "*")
	if err := r.store.DeleteAll(itemKeys); err != nil {
		return err
	}
	return r.store.Delete(redisLruKeyCacheKey)
}

// NewWriter instanciates a new writer
func (r *Redis) NewWriter(itemID string) io.WriteCloser {
	return &redis.Writer{
		ReadWrite: redis.ReadWrite{
			Store:     r.store,
			ItemID:    itemID,
			PrefixKey: redisLruItemCacheKey,
			UsageKey:  redisLruKeyCacheKey,
		},
	}
}

// NewReader instanciates a new reader
func (r *Redis) NewReader(itemID string, format sdk.CDNReaderFormat, from int64, size uint, sort int64) io.ReadCloser {
	return &redis.Reader{
		ReadWrite: redis.ReadWrite{
			Store:     r.store,
			ItemID:    itemID,
			PrefixKey: redisLruItemCacheKey,
			UsageKey:  redisLruKeyCacheKey,
		},
		Size:   size,
		From:   from,
		Format: format,
		Sort:   sort,
	}
}

// Evict evicts each 15s old entries
func (r *Redis) Evict(ctx context.Context) {
	tick := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "cdn:lru:evict: %v", ctx.Err())
			}
			return
		case <-tick.C:
			for {
				loop, err := r.eviction()
				if err != nil {
					log.Error(ctx, "cdn:lru:evict: %v", err)
					break
				}
				if !loop {
					break
				}
			}
		}
	}
}

func (r *Redis) eviction() (bool, error) {
	lenght, err := r.Len()
	if err != nil {
		return false, err
	}
	if lenght == 0 {
		return false, nil
	}
	size, err := r.Size()
	if err != nil {
		return false, err
	}
	if size <= r.MaxSize() {
		return false, err
	}
	if err := r.RemoveOldest(); err != nil {
		return false, err
	}
	return true, nil
}

// Get a value from the cache + update last usage
func (r *Redis) get(itemID string, from, to uint) ([]string, error) {
	var res = make([]string, to-from+1)
	itemKey := cache.Key(redisLruItemCacheKey, itemID)
	if err := r.store.ScoredSetScan(context.Background(), itemKey, float64(from), float64(to), &res); err != nil {
		return res, err
	}
	return res, nil
}

// Status returns the monitoring status
func (r *Redis) Status(ctx context.Context) []sdk.MonitoringStatusLine {
	if err := r.store.Ping(); err != nil {
		return []sdk.MonitoringStatusLine{{
			Component: "cache/log/ping",
			Value:     "connect KO",
			Status:    sdk.MonitoringStatusAlert,
		}}
	}

	dbsize, err := r.store.DBSize()
	if err != nil {
		return []sdk.MonitoringStatusLine{{
			Component: "cache/log/redis_dbsize",
			Value:     fmt.Sprintf("ERROR while getting cache log db size: %v err:%v", dbsize, err),
			Status:    sdk.MonitoringStatusAlert,
		}}
	}
	size, err := r.Size()
	if err != nil {
		return []sdk.MonitoringStatusLine{{
			Component: "cache/log/size",
			Value:     fmt.Sprintf("ERROR while getting cache log size: %v: err:%v", size, err),
			Status:    sdk.MonitoringStatusAlert,
		}}
	}
	len, err := r.Len()
	if err != nil {
		return []sdk.MonitoringStatusLine{{
			Component: "cache/log/nb",
			Value:     fmt.Sprintf("ERROR while getting cache log nb elements: %v err:%v", size, err),
			Status:    sdk.MonitoringStatusAlert,
		}}
	}

	statusSize := sdk.MonitoringStatusOK
	// if size is > 10Mo than maxSize -> Warn
	if r.maxSize-size < -10000000 {
		statusSize = sdk.MonitoringStatusWarn
	}
	// if size is > 20Mo than maxSize -> Warn
	if r.maxSize-size < -20000000 {
		statusSize = sdk.MonitoringStatusAlert
	}

	return []sdk.MonitoringStatusLine{
		{
			Component: "cache/log/redis_dbsize",
			Value:     fmt.Sprintf("%d keys", dbsize),
			Status:    sdk.MonitoringStatusOK,
		},
		{
			Component: "cache/log/ping",
			Value:     "connect OK",
			Status:    sdk.MonitoringStatusOK,
		},
		{
			Component: "cache/log/items",
			Value:     fmt.Sprintf("%d", len),
			Status:    sdk.MonitoringStatusOK,
		}, {
			Component: "cache/log/maxsize",
			Value:     fmt.Sprintf("%d octets", r.maxSize),
			Status:    sdk.MonitoringStatusOK,
		},
		{
			Component: "cache/log/size",
			Value:     fmt.Sprintf("%d octets", size),
			Status:    statusSize,
		},
	}
}
