package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
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
			MasterName:         masterName,
			SentinelAddrs:      sentinels,
			Password:           password,
			IdleCheckFrequency: 10 * time.Second,
			IdleTimeout:        10 * time.Second,
			PoolSize:           25,
		}
		client = redis.NewFailoverClient(opts)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:               host,
			Password:           password, // no password set
			DB:                 0,        // use default DB
			IdleCheckFrequency: 30 * time.Second,
		})
	}

	redis.SetLogger(stdlog.New(ioutil.Discard, "", stdlog.LstdFlags|stdlog.Lshortfile))

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

const retryWait = 30
const retryWaitDuration = retryWait * time.Millisecond

//Get a key from redis
func (s *RedisStore) Get(key string, value interface{}) bool {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return false
	}

	var errRedis error
	for i := 0; i < 3; i++ {
		val, errRedis := s.Client.Get(key).Result()
		if errRedis != nil && errRedis != redis.Nil {
			time.Sleep(retryWaitDuration)
			continue
		}
		if val != "" && errRedis != redis.Nil {
			if err := json.Unmarshal([]byte(val), value); err != nil {
				log.Warning("redis> cannot unmarshal %s :%s", key, err)
				return false
			}
			return true
		}
	}

	if errRedis != nil && errRedis != redis.Nil {
		log.Error("redis> get error %s : %v", key, errRedis)
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
		log.Warning("redis> error caching %s: %s", key, err)
	}

	var errRedis error
	for i := 0; i < 3; i++ {
		errRedis = s.Client.Set(key, string(b), time.Duration(ttl)*time.Second).Err()
		if errRedis == nil {
			break
		}
		time.Sleep(retryWaitDuration)
	}
	if errRedis != nil {
		log.Error("redis> set error %s: %v", key, errRedis)
	}
}

//UpdateTTL update the ttl linked to the key
func (s *RedisStore) UpdateTTL(key string, ttl int) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return
	}

	var errRedis error
	for i := 0; i < 3; i++ {
		errRedis = s.Client.Expire(key, time.Duration(ttl)*time.Second).Err()
		if errRedis == nil {
			break
		}
		time.Sleep(retryWaitDuration)
	}
	if errRedis != nil {
		log.Error("redis> set error %s: %v", key, errRedis)
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
	var errRedis error
	for i := 0; i < 3; i++ {
		errRedis = s.Client.Del(key).Err()
		if errRedis == nil {
			break
		}
		time.Sleep(retryWaitDuration)
	}
	if errRedis != nil {
		log.Error("redis> error deleting %s : %s", key, errRedis)
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

	var errRedis error
	var res int64
	for i := 0; i < 3; i++ {
		res, errRedis = s.Client.LLen(queueName).Result()
		if errRedis == nil {
			break
		}
		time.Sleep(retryWaitDuration)
	}
	if errRedis != nil {
		log.Warning("redis> Cannot read %s :%s", queueName, errRedis)
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
			res, err := s.Client.BRPop(time.Second, queueName).Result()
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
		if err := json.Unmarshal(b, value); err != nil {
			log.Error("redis.DequeueWithContext> error on unmarshal value on queue:%s err:%v", queueName, err)
		}
	}
}

// Publish a msg in a channel
func (s *RedisStore) Publish(channel string, value interface{}) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return
	}

	msg, err := json.Marshal(value)
	if err != nil {
		log.Warning("redis.Publish> Marshall error, cannot push in channel %s: %v", channel, err)
		return
	}

	iUnquoted, err := strconv.Unquote(string(msg))
	if err != nil {
		log.Warning("redis.Publish> Unquote error, cannot push in channel %s: %v", channel, err)
		return
	}

	for i := 0; i < 10; i++ {
		_, errP := s.Client.Publish(channel, iUnquoted).Result()
		if errP == nil {
			break
		}
		log.Warning("redis.Publish> Unable to publish in channel %s: %v", channel, errP)
		time.Sleep(100 * time.Millisecond)
	}
}

// Subscribe to a channel
func (s *RedisStore) Subscribe(channel string) PubSub {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return nil
	}
	return s.Client.Subscribe(channel)
}

// GetMessageFromSubscription from a redis PubSub
func (s *RedisStore) GetMessageFromSubscription(c context.Context, pb PubSub) (string, error) {
	if s.Client == nil {
		log.Error("redis> cannot get redis client")
		return "", nil
	}

	rps, ok := pb.(*redis.PubSub)
	if !ok {
		return "", fmt.Errorf("redis.GetMessage> PubSub is not a redis.PubSub. Got %T", pb)
	}

	msg, _ := rps.ReceiveTimeout(time.Second)
	redisMsg, ok := msg.(*redis.Message)
	if msg != nil {
		if ok {
			return redisMsg.Payload, nil
		}
	}

	ticker := time.NewTicker(250 * time.Millisecond).C
	for redisMsg == nil {
		select {
		case <-ticker:
			msg, _ := rps.ReceiveTimeout(time.Second)
			if msg == nil {
				continue
			}

			var ok bool
			redisMsg, ok = msg.(*redis.Message)
			if !ok {
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
		return sdk.MonitoringStatusLine{Component: "Cache Ping", Value: "OK", Status: sdk.MonitoringStatusOK}
	}
	return sdk.MonitoringStatusLine{Component: "Cache Ping", Value: "KO", Status: sdk.MonitoringStatusAlert}
}

// RemoveFromQueue removes a member from a list
func (s *RedisStore) RemoveFromQueue(rootKey string, memberKey string) {
	s.Client.LRem(rootKey, 0, memberKey)
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

	keys := make([]string, len(values))
	for i, v := range values {
		keys[i] = Key(key, v)
	}

	if len(keys) > 0 {
		res, err := s.Client.MGet(keys...).Result()
		if err != nil {
			return sdk.WrapError(err, "redis mget error")
		}

		for i := range members {
			if i >= len(values) {
				break
			}

			if res[i] == nil {
				//If the member is not found, return an error because the members are inconsistents
				// but try to delete the member from the Redis ZSET
				log.Error("redis>SetScan member %s not found", keys[i])
				if err := s.Client.ZRem(key, values[i]).Err(); err != nil {
					log.Error("redis>SetScan unable to delete member %s", keys[i])
					return err
				}
				log.Info("redis> member %s deleted", keys[i])
				return fmt.Errorf("SetScan member %s not found", keys[i])
			}

			if err := json.Unmarshal([]byte(res[i].(string)), members[i]); err != nil {
				log.Warning("redis> cannot unmarshal %s :%s", keys[i], err)
				return err
			}
		}
	}
	return nil
}

func (s *RedisStore) Lock(key string, expiration time.Duration, retrywdMillisecond int, retryCount int) bool {
	var errRedis error
	var res bool
	if retrywdMillisecond == -1 {
		retrywdMillisecond = retryWait
	}
	if retryCount == -1 {
		retryCount = 3
	}
	for i := 0; i < retryCount; i++ {
		res, errRedis = s.Client.SetNX(key, "true", expiration).Result()
		if errRedis == nil && res {
			break
		}
		time.Sleep(time.Duration(retrywdMillisecond) * time.Millisecond)
	}
	if errRedis != nil {
		log.Error("redis> set error %s: %v", key, errRedis)
	}
	return res
}

func (s *RedisStore) Unlock(key string) {
	s.Delete(key)
}
