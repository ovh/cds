package cache

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
)

// PubSub represents a subscriber
type PubSub interface {
	Unsubscribe(channels ...string) error
}

//Key make a key as expected
func Key(args ...string) string {
	return strings.Join(args, ":")
}

//Store is an interface
type Store interface {
	Get(key string, value interface{}) bool
	Set(key string, value interface{})
	SetWithTTL(key string, value interface{}, ttl int)
	UpdateTTL(key string, ttl int) error
	Delete(key string)
	DeleteAll(key string)
	Enqueue(queueName string, value interface{}) error
	DequeueWithContext(c context.Context, queueName string, value interface{}) error
	QueueLen(queueName string) (int, error)
	RemoveFromQueue(queueName string, memberKey string) error
	Publish(queueName string, value interface{}) error
	Subscribe(queueName string) (PubSub, error)
	GetMessageFromSubscription(c context.Context, pb PubSub) (string, error)
	Status() sdk.MonitoringStatusLine
	SetAdd(rootKey string, memberKey string, member interface{}) error
	SetRemove(rootKey string, memberKey string, member interface{}) error
	SetCard(key string) (int, error)
	SetScan(key string, members ...interface{}) error
	ZScan(key, pattern string) ([]string, error)
	Lock(key string, expiration time.Duration, retryWaitDurationMillisecond int, retryCount int) (bool, error)
	Unlock(key string) error
}

//New init a cache
func New(redisHost, redisPassword string, TTL int) (Store, error) {
	return NewRedisStore(redisHost, redisPassword, TTL)
}

//NewWriteCloser returns a write closer
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
	w.store.SetWithTTL(w.key, w.String(), w.ttl)
	return nil
}
