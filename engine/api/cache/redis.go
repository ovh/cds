package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ovh/cds/sdk/log"

	"gopkg.in/redis.v4"
)

//RedisStore a redis client and a default ttl
type RedisStore struct {
	ttl    int
	Client redisClient
}

//NewRedisStore initiate a new redisStore
func NewRedisStore(host, password string, ttl int) (*RedisStore, error) {
	var client redisClient

	//if host is line master@localhost:26379,localhost:26380 => it's a redis sentinel cluster
	if strings.Contains(host, "@") && strings.Contains(host, ",") {
		masterName := strings.Split(host, "@")[0]
		sentinelsStr := strings.Split(host, "@")[1]
		sentinels := strings.Split(sentinelsStr, ",")
		opts := &redis.FailoverOptions{
			MasterName:    masterName,
			SentinelAddrs: sentinels,
			Password:      password,
		}
		client = redis.NewFailoverClient(opts)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     host,
			Password: password, // no password set
			DB:       0,        // use default DB
		})
	}
	pong, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}
	if pong != "PONG" {
		return nil, fmt.Errorf("Cannot ping Redis on %s", host)
	}
	return &RedisStore{
		ttl:    ttl,
		Client: client,
	}, nil
}

//Get a key from redis
func (s *RedisStore) Get(key string, value interface{}) bool {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return false
	}
	val, err := s.Client.Get(key).Result()
	if err != nil && err != redis.Nil {
		log.Warning("redis> Get error %s : %s", key, err)
		return false
	}
	if val != "" && err != redis.Nil {
		if err := json.Unmarshal([]byte(val), value); err != nil {
			log.Warning("redis> Cannot unmarshal %s :%s", key, err)
			return false
		}
		return true
	}
	return false
}

//SetWithTTL a value in local store (0 for eternity)
func (s *RedisStore) SetWithTTL(key string, value interface{}, ttl int) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return
	}
	b, err := json.Marshal(value)
	if err != nil {
		log.Warning("redis> Error caching %s", key)
	}
	if err := s.Client.Set(key, string(b), time.Duration(ttl)*time.Second).Err(); err != nil {
		log.Warning("redis> Error caching %s", key)
	}
}

//Set a value in redis
func (s *RedisStore) Set(key string, value interface{}) {
	s.SetWithTTL(key, value, s.ttl)
}

//Delete a key in redis
func (s *RedisStore) Delete(key string) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return
	}
	if err := s.Client.Del(key).Err(); err != nil {
		log.Warning("redis> Error deleting %s : %s", key, err)
	}
}

//DeleteAll delete all mathing keys in redis
func (s *RedisStore) DeleteAll(pattern string) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return
	}
	keys, err := s.Client.Keys(pattern).Result()
	if err != nil {
		log.Warning("redis> Error deleting %s : %s", pattern, err)
		return
	}
	if len(keys) == 0 {
		return
	}
	if err := s.Client.Del(keys...).Err(); err != nil {
		log.Warning("redis> Error deleting %s : %s", pattern, err)
	}
}

//Enqueue pushes to queue
func (s *RedisStore) Enqueue(queueName string, value interface{}) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return
	}
	b, err := json.Marshal(value)
	if err != nil {
		log.Warning("redis> Error queueing %s:%s", queueName, err)
	}
	if err := s.Client.LPush(queueName, string(b)).Err(); err != nil {
		log.Warning("redis> Error while LPUSH to %s: %s", queueName, err)
	}
}

//Dequeue gets from queue This is blocking while there is nothing in the queue
func (s *RedisStore) Dequeue(queueName string, value interface{}) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return
	}
read:
	res, err := s.Client.BRPop(0, queueName).Result()
	if err != nil {
		log.Warning("redis> Error dequeueing %s:%s", queueName, err)
		if err == io.EOF {
			time.Sleep(1 * time.Second)
			goto read
		}
	}
	if len(res) != 2 {
		return
	}
	if err := json.Unmarshal([]byte(res[1]), value); err != nil {
		log.Warning("redis> Cannot unmarshal %s :%s", queueName, err)
	}
}

func (s *RedisStore) DequeueWithContext(queueName string, value interface{}, c context.Context) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return
	}

	elemChan := make(chan string)
	var once sync.Once
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond).C
		for {
			select {
			case <-ticker:
				res, err := s.Client.RPop(queueName).Result()
				if err == redis.Nil {
					continue
				}
				if err == io.EOF {
					time.Sleep(1 * time.Second)
					continue
				}
				if len(res) != 2 {
					continue
				}
				elemChan <- res
			case <-c.Done():
				once.Do(func() {
					close(elemChan)
				})
				return
			}
		}
	}()

	e := <-elemChan
	if e != "" {
		b := []byte(e)
		json.Unmarshal(b, value)
	}
	once.Do(func() {
		close(elemChan)
	})
}
