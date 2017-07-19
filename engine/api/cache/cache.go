package cache

import (
	"container/list"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ovh/cds/sdk/log"
)

//Status : local ok redis
var Status string

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
}

//Initialize the global cache in memory, or redis
func Initialize(mode, redisHost, redisPassword string, TTL int) {
	Status = mode
	switch mode {
	case "local":
		log.Info("Cache> Initialize local cache (TTL=%d seconds)", TTL)
		s = &LocalStore{
			Mutex:  &sync.Mutex{},
			Data:   map[string][]byte{},
			Queues: map[string]*list.List{},
			TTL:    TTL,
		}
	case "redis":
		log.Info("Cache> Initialize redis cache (Host=%s, TTL=%d seconds)", redisHost, TTL)
		var err error
		s, err = NewRedisStore(redisHost, redisPassword, TTL)
		if err != nil {
			Status += " KO"
			log.Error("cache> Cannot init redis cache (Host=%s, TTL=%d seconds): %s", redisHost, TTL, err)
		}
		Status += " OK"
	default:
		log.Error("Cache> Unsupported cache mode : %s", mode)
		Status = "KO"
	}
}

//Get something from the cache.
func Get(key string, value interface{}) bool {
	if s == nil {
		return false
	}
	return s.Get(key, value)
}

//Set something from the cache.
func Set(key string, value interface{}) {
	if s == nil {
		return
	}
	s.Set(key, value)
}

//SetWithTTL something in the cache with a specific TTL (second).
func SetWithTTL(key string, value interface{}, ttl int) {
	if s == nil {
		return
	}
	s.SetWithTTL(key, value, ttl)
}

//Delete something from the cache.
func Delete(key string) {
	if s == nil {
		return
	}
	s.Delete(key)
}

//DeleteAll something from the cache.
func DeleteAll(key string) {
	if s == nil {
		return
	}
	s.DeleteAll(key)
}

//Enqueue pushes to queue
func Enqueue(queueName string, value interface{}) {
	if s == nil {
		return
	}
	s.Enqueue(queueName, value)
}

//Dequeue gets from queue This is blocking while there is nothing in the queue
func Dequeue(queueName string, value interface{}) {
	if s == nil {
		return
	}
	s.Dequeue(queueName, value)
}

//DequeueWithContext gets from queue This is blocking while there is nothing in the queue, it can be cancelled with a context.Context
func DequeueWithContext(c context.Context, queueName string, value interface{}) {
	if s == nil {
		return
	}
	s.DequeueWithContext(c, queueName, value)
}

//QueueLen returns the length of a queue
func QueueLen(queueName string) int {
	if s == nil {
		return 0
	}
	return s.QueueLen(queueName)
}

// Publish a message on a channel
func Publish(queueName string, value interface{}) {
	if s == nil {
		return
	}
	s.Publish(queueName, value)
}

// Subscribe to a channel
func Subscribe(queueName string) PubSub {
	if s == nil {
		return nil
	}
	return s.Subscribe(queueName)
}

// GetMessageFromSubscription Get a message from a subscription
func GetMessageFromSubscription(pb PubSub, c context.Context) (string, error) {
	if s == nil {
		return "", fmt.Errorf("Cache > Client store is nil")
	}
	return s.GetMessageFromSubscription(c, pb)
}
