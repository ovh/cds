package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

// PubSub represents a subscriber
type PubSub interface {
	Unsubscribe(ctx context.Context, channels ...string) error
	GetMessage(ctx context.Context) (string, error)
}

// Key make a key as expected
func Key(args ...string) string {
	return strings.Join(args, ":")
}

// Store is an interface
type Store interface {
	Get(ctx context.Context, key string, value interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}) error
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl int) error
	SetWithDuration(ctx context.Context, key string, value interface{}, duration time.Duration) error
	UpdateTTL(ctx context.Context, key string, ttl int) error
	Delete(ctx context.Context, key string) error
	DeleteAll(ctx context.Context, key string) error
	Exist(ctx context.Context, key string) (bool, error)
	Eval(ctx context.Context, expr string, args ...string) (string, error)
	HealthStore
	LockStore
	QueueStore
	PubSubStore
	ScoredSetStore
	SetStore
}

type HealthStore interface {
	Ping(ctx context.Context) error
	DBSize(ctx context.Context) (int64, error)
	Size(ctx context.Context, key string) (int64, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
}

type QueueStore interface {
	Enqueue(ctx context.Context, queueName string, value interface{}) error
	DequeueWithContext(ctx context.Context, queueName string, waitDuration time.Duration, value interface{}) error
	DequeueJSONRawMessagesWithContext(ctx context.Context, queueName string, waitDuration time.Duration, maxElements int) ([]json.RawMessage, error)
	QueueLen(ctx context.Context, queueName string) (int, error)
	RemoveFromQueue(ctx context.Context, queueName string, memberKey string) error
}

type SetStore interface {
	SetAdd(ctx context.Context, rootKey string, memberKey string, member interface{}) error
	SetRemove(ctx context.Context, rootKey string, memberKey string, member interface{}) error
	SetCard(ctx context.Context, key string) (int, error)
	SetScan(ctx context.Context, key string, members ...interface{}) error
	SetSearch(ctx context.Context, key, pattern string) ([]string, error)
}

type PubSubStore interface {
	Publish(ctx context.Context, queueName string, value interface{}) error
	Subscribe(ctx context.Context, queueName string) (PubSub, error)
}

type LockStore interface {
	Lock(ctx context.Context, key string, expiration time.Duration, retryWaitDurationMillisecond int, retryCount int) (bool, error)
	Unlock(ctx context.Context, key string) error
}

type ScoredSetStore interface {
	Delete(ctx context.Context, key string) error
	ScoredSetAdd(ctx context.Context, key string, value interface{}, score float64) error
	ScoredSetAppend(ctx context.Context, key string, value interface{}) error
	ScoredSetScan(ctx context.Context, key string, from, to float64, dest interface{}) error
	ScoredSetScanWithScores(ctx context.Context, key string, from, to float64) ([]SetValueWithScore, error)
	ScoredSetScanMaxScore(ctx context.Context, key string) (*SetValueWithScore, error)
	ScoredSetRange(ctx context.Context, key string, from, to int64, dest interface{}) error
	ScoredSetRevRange(_ context.Context, key string, offset int64, limit int64, dest interface{}) error
	ScoredSetRem(ctx context.Context, key string, members ...string) error
	ScoredSetGetScore(ctx context.Context, key string, member interface{}) (float64, error)
	SetCard(ctx context.Context, key string) (int, error)
	Eval(ctx context.Context, expr string, args ...string) (string, error)
	HealthStore
}

type SetValueWithScore struct {
	Score float64
	Value json.RawMessage
}

// New init a cache
func New(ctx context.Context, redisConf sdk.RedisConf, TTL int) (Store, error) {
	return NewRedisStore(ctx, redisConf, TTL)
}

// NewWriteCloser returns a write closer
func NewWriteCloser(store Store, key string, ttl int) io.WriteCloser {
	return &writerCloser{
		store: store,
		key:   key,
		ttl:   ttl,
	}
}

type writerCloser struct {
	store Store
	key   string
	ttl   int
	bytes.Buffer
}

func (w *writerCloser) Close() error {
	if err := w.store.SetWithTTL(context.Background(), w.key, w.String(), w.ttl); err != nil {
		log.Error(context.TODO(), "cannot SetWithTTL: %s: %v", w.key, err)
	}
	return nil
}
