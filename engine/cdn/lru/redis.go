package lru

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/index"
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

func (r *Redis) UpdateUsage(itemID string) error {
	return r.store.ScoredSetAdd(context.Background(), redisLruKeyCacheKey, itemID, float64(time.Now().UnixNano()))
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
	return &writer{redis: r, itemID: itemID}
}

// NewReader
func (r *Redis) NewReader(itemID string, from uint, s int) io.ReadCloser {
	return &reader{redis: r, itemID: itemID, from: from, size: s - 1}
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
	if size <= r.MaxSize() {
		return false, err
	}
	if err := r.RemoveOldest(); err != nil {
		return false, err
	}
	return true, nil
}

// Add new item in cache + update last usage
func (r *Redis) add(itemID string, score uint, value string) error {
	itemKey := cache.Key(redisLruItemCacheKey, itemID)
	if err := r.store.ScoredSetAdd(context.Background(), itemKey, value, float64(score)); err != nil {
		return err
	}
	return nil
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

type writer struct {
	redis        *Redis
	itemID       string
	currentScore uint
}

type reader struct {
	redis         *Redis
	itemID        string
	lastIndex     uint
	from          uint
	size          int
	currentBuffer string
}

func (r *reader) Read(p []byte) (n int, err error) {
	size := len(p)
	var buffer string
	if len(r.currentBuffer) > 0 {
		if len(r.currentBuffer) <= size {
			buffer = r.currentBuffer
		}
	}

	var newFromIndex uint
	var newToIndex uint

	// First read
	if r.from > 0 && r.lastIndex == 0 {
		r.lastIndex = r.from
	}
	if size >= 0 && newFromIndex+100 > (r.from+uint(r.size)) {
		newToIndex = r.from + uint(r.size)
	} else {
		newToIndex = newFromIndex + 100
	}

	lines, err := r.redis.get(r.itemID, r.lastIndex, newToIndex)
	if err != nil {
		return 0, err
	}

	if len(lines) > 0 {
		r.currentBuffer += strings.Join(lines, "")
	}

	if len(buffer) < size && len(r.currentBuffer) > 0 {
		x := size - len(buffer)
		if x < len(r.currentBuffer) {
			buffer += r.currentBuffer[:x]
			r.currentBuffer = r.currentBuffer[x:]
		} else {
			buffer += r.currentBuffer
			r.currentBuffer = ""
		}
	}

	r.lastIndex = newToIndex
	err = nil
	if len(lines) == 0 || (r.size >= 0 && r.lastIndex == (r.from+uint(r.size))) {
		err = io.EOF
	}

	return copy(p, buffer), err
}

func (r *reader) Close() error {
	if err := r.redis.UpdateUsage(r.itemID); err != nil {
		return err
	}
	return nil
}

func (w *writer) Write(p []byte) (int, error) {
	// Get data at the current score
	lines, err := w.redis.get(w.itemID, w.currentScore, w.currentScore)
	if err != nil {
		return 0, err
	}
	var currentLine string
	if len(lines) == 1 {
		currentLine = lines[0]
	}

	var n int

	for _, bch := range p {
		charact := string(bch)
		currentLine = currentLine + charact
		n++
		if charact == "\n" {
			if err := w.redis.add(w.itemID, w.currentScore, currentLine); err != nil {
				return 0, err
			}
			w.currentScore++
			currentLine = ""
		}
	}

	// Save into redis current non-finished line
	if len(currentLine) > 0 {
		if err := w.redis.add(w.itemID, w.currentScore, currentLine); err != nil {
			return 0, err
		}
	}

	return n, nil
}

func (w *writer) Close() error {
	// Update last usage
	if err := w.redis.UpdateUsage(w.itemID); err != nil {
		return err
	}
	return nil
}
