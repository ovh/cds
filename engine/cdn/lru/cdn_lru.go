package lru

import (
	"context"
	"io"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk/log"
)

type AbstractLRU struct {
	maxSize int64
	db      *gorp.DbMap
}

type Interface interface {
	Exist(itemID string) (bool, error)
	NewReader(itemID string, from, int uint) io.ReadCloser
	NewWriter(itemID string) io.WriteCloser
	UpdateUsage(itemID string) error
	Remove(itemID string) error
	RemoveOldest() error

	Len() (int, error)
	Size() (int64, error)
	MaxSize() int64
	Clear() error
}

//New init a cache
func NewLRU(db *gorp.DbMap, redisHost string, redisPassword string, size int64) (Interface, error) {
	store, err := cache.New(redisHost, redisPassword, -1)
	if err != nil {
		return nil, err
	}
	a := AbstractLRU{
		db:      db,
		maxSize: size,
	}
	r := NewRedisLRU(a, store)
	return r, nil
}

func Evict(ctx context.Context, lru Interface) {
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
				loop, err := eviction(lru)
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

func eviction(lru Interface) (bool, error) {
	len, err := lru.Len()
	if err != nil {
		return false, err
	}
	if len == 0 {
		return false, nil
	}
	size, err := lru.Size()
	if err != nil {
		return false, err
	}
	if size <= lru.MaxSize() {
		return false, err
	}
	if err := lru.RemoveOldest(); err != nil {
		return false, err
	}
	return true, nil
}
