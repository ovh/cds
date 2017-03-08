package cache

import (
	"container/list"
	"strings"
	"sync"

	"github.com/ovh/cds/engine/log"
)

//Status : local ok redis
var Status string

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
}

//Initialize the global cache in memory, or redis
func Initialize(mode, redisHost, redisPassword string, TTL int) {
	Status = mode
	switch mode {
	case "local":
		log.Notice("Cache> Initialize local cache (TTL=%d seconds)", TTL)
		s = &LocalStore{
			Mutex:  &sync.Mutex{},
			Data:   map[string][]byte{},
			Queues: map[string]*list.List{},
			TTL:    TTL,
		}
	case "redis":
		log.Notice("Cache> Initialize redis cache (Host=%s, TTL=%d seconds)", redisHost, TTL)
		var err error
		s, err = NewRedisStore(redisHost, redisPassword, TTL)
		if err != nil {
			Status += " KO"
			log.Critical("cache> Cannot init redis cache (Host=%s, TTL=%d seconds): %s", redisHost, TTL, err)
		}
		Status += " OK"
	default:
		log.Critical("Cache> Unsupported cache mode : %s", mode)
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
