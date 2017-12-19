package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//RedisStore a redis client and a default ttl
type RedisStore struct {
	ttl    int
	Client *redis.Client
}

//NewRedisStore initiate a new redisStore
func NewRedisStore(host, password string, ttl int) (*RedisStore, error) {
	var client *redis.Client

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
		log.Warning("redis> Get error %s : %v", key, err)
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
		log.Warning("redis> Error caching %s: %s", key, err)
	}
	if err := s.Client.Set(key, string(b), time.Duration(ttl)*time.Second).Err(); err != nil {
		log.Warning("redis> Error caching %s: %s", key, err)
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

//QueueLen returns the length of a queue
func (s *RedisStore) QueueLen(queueName string) int {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return 0
	}

	res, err := s.Client.LLen(queueName).Result()
	if err != nil {
		log.Warning("redis> Cannot read %s :%s", queueName, err)
	}
	return int(res)
}

//DequeueWithContext gets from queue This is blocking while there is nothing in the queue, it can be cancelled with a context.Context
func (s *RedisStore) DequeueWithContext(c context.Context, queueName string, value interface{}) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return
	}

	var elem string
	ticker := time.NewTicker(250 * time.Millisecond).C
	for elem == "" {
		select {
		case <-ticker:
			res, err := s.Client.BRPop(200*time.Millisecond, queueName).Result()
			if err == redis.Nil {
				continue
			}
			if err == io.EOF {
				time.Sleep(1 * time.Second)
				continue
			}
			if err == nil && len(res) == 2 {
				elem = res[1]
				break
			}
		case <-c.Done():
			return
		}
	}
	if elem != "" {
		b := []byte(elem)
		json.Unmarshal(b, value)
	}
}

// Publish a msg in a channel
func (s *RedisStore) Publish(channel string, value interface{}) {
	msg, err := json.Marshal(value)
	if err != nil {
		log.Warning("redis.Publish> Marshall error, cannot push in channel %s: %v, %s", channel, value, err)
		return
	}
	iUnquoted, err := strconv.Unquote(string(msg))

	if err != nil {
		log.Warning("redis.Publish> Unquote error, cannot push in channel %s: %v, %s", channel, string(msg), err)
		return
	}

	s.Client.Publish(channel, iUnquoted)
}

// Subscribe to a channel
func (s *RedisStore) Subscribe(channel string) PubSub {
	return s.Client.Subscribe(channel)
}

// GetMessageFromSubscription from a redis PubSub
func (s *RedisStore) GetMessageFromSubscription(c context.Context, pb PubSub) (string, error) {
	rps, ok := pb.(*redis.PubSub)
	if !ok {
		return "", fmt.Errorf("redis.GetMessage> PubSub is not a redis.PubSub. Got %T", pb)
	}

	msg, _ := rps.ReceiveTimeout(200 * time.Millisecond)
	redisMsg, ok := msg.(*redis.Message)
	if msg != nil {
		if ok {
			return redisMsg.Payload, nil
		}
		log.Warning("redis.GetMessage> Message casting error for %v of type %T", msg, msg)
	}

	ticker := time.NewTicker(250 * time.Millisecond).C
	for redisMsg == nil {
		select {
		case <-ticker:
			msg, _ := rps.ReceiveTimeout(200 * time.Millisecond)
			if msg == nil {
				continue
			}

			var ok bool
			redisMsg, ok = msg.(*redis.Message)
			if !ok {
				log.Warning("redis.GetMessage> Message casting error for %v of type %T", msg, msg)
				continue
			}
		case <-c.Done():
			return "", nil
		}
	}
	return redisMsg.Payload, nil
}

// Status returns the status of the local cache
func (s *RedisStore) Status() sdk.MonitoringStatusLine {
	if s.Client.Ping().Err() == nil {
		return sdk.MonitoringStatusLine{Component: "Cache", Value: "Ping OK", Status: sdk.MonitoringStatusOK}
	}
	return sdk.MonitoringStatusLine{Component: "Cache", Value: "No Ping", Status: sdk.MonitoringStatusAlert}
}

// SetAdd add a member (identified by a key) in the cached set
func (s *RedisStore) SetAdd(rootKey string, memberKey string, member interface{}) {
	s.Client.ZAdd(rootKey, redis.Z{
		Member: memberKey,
		Score:  float64(time.Now().UnixNano()),
	})
	s.SetWithTTL(Key(rootKey, memberKey), member, -1)
}

// SetRemove removes a member from a set
func (s *RedisStore) SetRemove(rootKey string, memberKey string, member interface{}) {
	s.Client.ZRem(rootKey, memberKey)
	s.Delete(Key(rootKey, memberKey))
}

// SetCard returns the cardinality of a ZSet
func (s *RedisStore) SetCard(key string) int {
	return int(s.Client.ZCard(key).Val())
}

// SetScan scans a ZSet
func (s *RedisStore) SetScan(key string, members ...interface{}) error {
	values, err := s.Client.ZRangeByScore(key, redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return sdk.WrapError(err, "redis zrange error")
	}

	for i := range members {
		if i >= len(values) {
			break
		}
		val := values[i]
		memKey := Key(key, val)
		if !s.Get(memKey, members[i]) {
			return fmt.Errorf("Member (%s) not found", memKey)
		}
	}
	return nil
}
