package cache

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"
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
	SetWithDuration(key string, value interface{}, duration time.Duration) error
	UpdateTTL(key string, ttl int)
	Delete(key string)
	DeleteAll(key string)
	Enqueue(queueName string, value interface{})
	Dequeue(queueName string, value interface{})
	DequeueWithContext(c context.Context, queueName string, value interface{})
	QueueLen(queueName string) int
	RemoveFromQueue(queueName string, memberKey string)
	Publish(queueName string, value interface{})
	Subscribe(queueName string) PubSub
	GetMessageFromSubscription(c context.Context, pb PubSub) (string, error)
	SetAdd(rootKey string, memberKey string, member interface{})
	SetRemove(rootKey string, memberKey string, member interface{})
	SetCard(key string) int
	SetScan(key string, members ...interface{}) error
	Lock(key string, expiration time.Duration, retryWaitDurationMillisecond int, retryCount int) bool
	Unlock(key string)
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
