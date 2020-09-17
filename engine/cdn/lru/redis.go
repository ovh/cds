package lru

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/index"
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

func NewRedisLRU(db *gorp.DbMap, maxSize int64, host string, password string) (*Redis, error) {
	c, err := cache.New(host, password, -1)
	if err != nil {
		return nil, err
	}
	return &Redis{db: db, maxSize: maxSize, store: c}, nil
}

func (r *Redis) Exist(itemID string) (bool, error) {
	itemKey := cache.Key(redisLruItemCacheKey, itemID)
	return r.store.Exist(itemKey)
}

func (r *Redis) Remove(itemID string) error {
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
	return nil
}

func (r *Redis) RemoveOldest() error {
	var keys []string
	if err := r.store.ScoredSetRange(context.Background(), redisLruKeyCacheKey, 0, 0, &keys); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return sdk.WithStack(r.Remove(keys[0]))
}

// Number of elements in that cache
func (r *Redis) Len() (int, error) {
	return r.store.SetCard(redisLruKeyCacheKey)
}

// Size of the cache
func (r *Redis) Size() (int64, error) {
	var itemIDs []string
	if err := r.store.ScoredSetRange(context.Background(), redisLruKeyCacheKey, 0, -1, &itemIDs); err != nil {
		return 0, err
	}

	// DB Request
	return index.ComputeSizeByItemIDs(r.db, itemIDs)
}

func (r *Redis) MaxSize() int64 {
	return r.maxSize
}

func (r *Redis) Clear() error {
	itemKeys := cache.Key(redisLruItemCacheKey, "*")
	if err := r.store.DeleteAll(itemKeys); err != nil {
		return err
	}
	return r.store.Delete(redisLruKeyCacheKey)
}

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

// NewReader
func (r *Redis) NewReader(itemID string, from uint, s int) io.ReadCloser {
	return &redis.Reader{
		ReadWrite: redis.ReadWrite{
			Store:     r.store,
			ItemID:    itemID,
			PrefixKey: redisLruItemCacheKey,
			UsageKey:  redisLruKeyCacheKey,
		},
		Size: s,
		From: from,
	}
}

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
	log.Debug("cdn:lru:  %d/%d", size, r.MaxSize())
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
