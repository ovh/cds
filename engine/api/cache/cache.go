package cache

import (
	"context"
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
	Status() sdk.MonitoringStatusLine
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
