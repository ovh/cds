package cache

import (
	"container/list"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ovh/cds/sdk/log"
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
	Publish(queueName string, value interface{})
	Subscribe(queueName string) PubSub
	GetMessageFromSubscription(c context.Context, pb PubSub) (string, error)
	Status() string
}

//New init a cache
func New(mode, redisHost, redisPassword string, TTL int) (Store, error) {
	log.Debug("New cache")
	switch mode {
	case "local":
		log.Info("Cache> Initialize local cache (TTL=%d seconds)", TTL)
		return &LocalStore{
			mutex:  &sync.Mutex{},
			Data:   map[string][]byte{},
			Queues: map[string]*list.List{},
			TTL:    TTL,
		}, nil
	case "redis":
		log.Info("Cache> Initialize redis cache (Host=%s, TTL=%d seconds)", redisHost, TTL)
		return NewRedisStore(redisHost, redisPassword, TTL)
	default:
		return nil, fmt.Errorf("Cache> Unsupported cache mode : %s", mode)
	}
}
