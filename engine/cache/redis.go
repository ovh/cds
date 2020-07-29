package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"math"
	"reflect"
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
			MaxRetries:         10,
			MinRetryBackoff:    30 * time.Millisecond,
			MaxRetryBackoff:    100 * time.Millisecond,
		}
		client = redis.NewFailoverClient(opts)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:               host,
			Password:           password, // no password set
			DB:                 0,        // use default DB
			IdleCheckFrequency: 30 * time.Second,
			MaxRetries:         10,
			MinRetryBackoff:    30 * time.Millisecond,
			MaxRetryBackoff:    100 * time.Millisecond,
		})
	}

	redis.SetLogger(stdlog.New(ioutil.Discard, "", stdlog.LstdFlags|stdlog.Lshortfile))

	pong, err := client.Ping().Result()
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if pong != "PONG" {
		return nil, fmt.Errorf("Cannot ping Redis on %s", host)
	}
	return &RedisStore{
		ttl:    ttl,
		Client: client,
	}, nil
}

// Keys List keys from pattern
func (s *RedisStore) Keys(pattern string) ([]string, error) {
	if s.Client == nil {
		return nil, sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}
	keys, err := s.Client.Keys(pattern).Result()
	if err != nil {
		return nil, sdk.WrapError(err, "redis> cannot list keys: %s", pattern)
	}
	return keys, nil
}

// Get a key from redis
func (s *RedisStore) Get(key string, value interface{}) (bool, error) {
	if s.Client == nil {
		return false, sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}

	val, errRedis := s.Client.Get(key).Result()
	if errRedis != nil && errRedis != redis.Nil {
		return false, sdk.WrapError(errRedis, "redis> get error %s", key)
	}
	if val != "" {
		if err := json.Unmarshal([]byte(val), value); err != nil {
			return false, sdk.WrapError(err, "redis> cannot get unmarshal %s", key)
		}
		return true, nil
	}

	return false, nil
}

// SetWithTTL a value in local store (0 for eternity)
func (s *RedisStore) SetWithTTL(key string, value interface{}, ttl int) error {
	if s.Client == nil {
		return sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}

	b, err := json.Marshal(value)
	if err != nil {
		return sdk.WrapError(err, "redis> error caching %s", key)
	}

	if err := s.Client.Set(key, string(b), time.Duration(ttl)*time.Second).Err(); err != nil {
		return sdk.WrapError(err, "redis> set error %s", key)
	}
	return nil
}

// SetWithDuration a value in local store (0 for eternity)
func (s *RedisStore) SetWithDuration(key string, value interface{}, duration time.Duration) error {
	if s.Client == nil {
		return nil
	}

	b, err := json.Marshal(value)
	if err != nil {
		return sdk.WithStack(err)
	}

	if err := s.Client.Set(key, string(b), duration).Err(); err != nil {
		return sdk.WrapError(err, "set error %s", key)
	}

	return nil
}

// UpdateTTL update the ttl linked to the key
func (s *RedisStore) UpdateTTL(key string, ttl int) error {
	if s.Client == nil {
		return sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}

	if err := s.Client.Expire(key, time.Duration(ttl)*time.Second).Err(); err != nil {
		return sdk.WrapError(err, "redis>UpdateTTL> set error %s", key)
	}
	return nil
}

// Set a value in redis
func (s *RedisStore) Set(key string, value interface{}) error {
	return s.SetWithTTL(key, value, s.ttl)
}

// Delete a key in redis
func (s *RedisStore) Delete(key string) error {
	if s.Client == nil {
		return sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}

	if err := s.Client.Del(key).Err(); err != nil {
		return sdk.WrapError(err, "redis> error deleting %s", key)
	}
	return nil
}

// DeleteAll delete all mathing keys in redis
func (s *RedisStore) DeleteAll(pattern string) error {
	if s.Client == nil {
		return sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}
	keys, err := s.Client.Keys(pattern).Result()
	if err != nil {
		return sdk.WrapError(err, "redis> Error deleting %s", pattern)
	}
	if len(keys) == 0 {
		return nil
	}
	if err := s.Client.Del(keys...).Err(); err != nil {
		return sdk.WrapError(err, "redis> Error deleting %s", pattern)
	}
	return nil
}

// Exist test is key exists
func (s *RedisStore) Exist(key string) (bool, error) {
	if s.Client == nil {
		return false, sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}
	ok, err := s.Client.Exists(key).Result()
	if err != nil {
		return false, sdk.WrapError(err, "unable to test if key %s exists", key)
	}
	return ok == 1, nil
}

// Enqueue pushes to queue
func (s *RedisStore) Enqueue(queueName string, value interface{}) error {
	if s.Client == nil {
		return sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}
	b, err := json.Marshal(value)
	if err != nil {
		return sdk.WrapError(err, "error queueing %s:%s", queueName, err)
	}
	if err := s.Client.LPush(queueName, string(b)).Err(); err != nil {
		return sdk.WrapError(err, "error while LPUSH to %s: %s", queueName, err)
	}
	return nil
}

// QueueLen returns the length of a queue
func (s *RedisStore) QueueLen(queueName string) (int, error) {
	if s.Client == nil {
		return 0, sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}
	var errRedis error
	var res int64
	res, errRedis = s.Client.LLen(queueName).Result()
	if errRedis != nil {
		return 0, sdk.WrapError(errRedis, "redis> Cannot read %s", queueName)
	}
	return int(res), nil
}

// DequeueWithContext gets from queue This is blocking while there is nothing in the queue, it can be cancelled with a context.Context
func (s *RedisStore) DequeueWithContext(c context.Context, queueName string, waitDuration time.Duration, value interface{}) error {
	if s.Client == nil {
		return sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}

	var elem string
	ticker := time.NewTicker(waitDuration)
	defer ticker.Stop()
	for elem == "" {
		select {
		case <-ticker.C:
			if c.Err() != nil {
				return c.Err()
			}
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
			return nil
		}
	}
	if elem != "" {
		b := []byte(elem)
		if err := json.Unmarshal(b, value); err != nil {
			return sdk.WrapError(err, "redis.DequeueWithContext> error on unmarshal value on queue:%s", queueName)
		}
	}
	return nil
}

// Publish a msg in a channel
func (s *RedisStore) Publish(ctx context.Context, channel string, value interface{}) error {
	if s.Client == nil {
		return sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}

	msg, err := json.Marshal(value)
	if err != nil {
		return sdk.WrapError(err, "redis.Publish> Marshall error, cannot push in channel %s", channel)
	}

	iUnquoted, err := strconv.Unquote(string(msg))
	if err != nil {
		return sdk.WrapError(err, "redis.Publish> Unquote error, cannot push in channel %s", channel)
	}

	for i := 0; i < 10; i++ {
		_, errP := s.Client.Publish(channel, iUnquoted).Result()
		if errP == nil {
			break
		}
		log.Warning(ctx, "redis.Publish> Unable to publish in channel %s: %v", channel, errP)
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// Subscribe to a channel
func (s *RedisStore) Subscribe(channel string) (PubSub, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("redis> cannot get redis client")
	}
	return s.Client.Subscribe(channel), nil
}

// GetMessageFromSubscription from a redis PubSub
func (s *RedisStore) GetMessageFromSubscription(c context.Context, pb PubSub) (string, error) {
	if s.Client == nil {
		return "", sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
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

// RemoveFromQueue removes a member from a list
func (s *RedisStore) RemoveFromQueue(rootKey string, memberKey string) error {
	if err := s.Client.LRem(rootKey, 0, memberKey).Err(); err != nil {
		return sdk.WrapError(err, "error on RemoveFromQueue: rooKey:%v memberKey:%v", rootKey, memberKey)
	}
	return nil
}

// SetAdd add a member (identified by a key) in the cached set
func (s *RedisStore) SetAdd(rootKey string, memberKey string, member interface{}) error {
	err := s.Client.ZAdd(rootKey, redis.Z{
		Member: memberKey,
		Score:  float64(time.Now().UnixNano()),
	}).Err()
	if err != nil {
		return sdk.WrapError(err, "error on SetAdd")
	}
	return s.SetWithTTL(Key(rootKey, memberKey), member, -1)
}

// SetRemove removes a member from a set
func (s *RedisStore) SetRemove(rootKey string, memberKey string, member interface{}) error {
	if err := s.Client.ZRem(rootKey, memberKey).Err(); err != nil {
		return sdk.WrapError(err, "error on SetRemove")
	}
	return s.Delete(Key(rootKey, memberKey))
}

// SetCard returns the cardinality of a ZSet
func (s *RedisStore) SetCard(key string) (int, error) {
	v := s.Client.ZCard(key)
	return int(v.Val()), v.Err()
}

// SetScan scans a ZSet
func (s *RedisStore) SetScan(ctx context.Context, key string, members ...interface{}) error {
	values, err := s.Client.ZRangeByScore(key, redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return fmt.Errorf("redis zrange error: %v", err)
	}

	keys := make([]string, len(values))
	for i, v := range values {
		keys[i] = Key(key, v)
	}

	if len(keys) > 0 {
		res, err := s.Client.MGet(keys...).Result()
		if err != nil {
			return fmt.Errorf("redis mget error: %v", err)
		}

		for i := range members {
			if i >= len(values) {
				break
			}

			if res[i] == nil {
				//If the member is not found, return an error because the members are inconsistents
				// but try to delete the member from the Redis ZSET
				log.Error(ctx, "redis>SetScan member %s not found", keys[i])
				if err := s.Client.ZRem(key, values[i]).Err(); err != nil {
					return sdk.WrapError(err, "redis>SetScan unable to delete member %s", keys[i])
				}
				log.Info(ctx, "redis> member %s deleted", keys[i])
				return sdk.WithStack(fmt.Errorf("SetScan member %s not found", keys[i]))
			}

			if err := json.Unmarshal([]byte(res[i].(string)), members[i]); err != nil {
				return sdk.WrapError(err, "redis> cannot unmarshal %s", keys[i])
			}
		}
	}
	return nil
}

func (s *RedisStore) SetSearch(key, pattern string) ([]string, error) {
	keys, _, err := s.Client.ZScan(key, 0, pattern, 0).Result()
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	return keys, nil
}

func (s *RedisStore) Lock(key string, expiration time.Duration, retrywdMillisecond int, retryCount int) (bool, error) {
	var errRedis error
	var res bool
	if retrywdMillisecond == -1 {
		retrywdMillisecond = 30
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
	return res, sdk.WrapError(errRedis, "redis> set error %s", key)
}

// Unlock deletes a key from cache
func (s *RedisStore) Unlock(key string) error {
	return s.Delete(key)
}

func (s *RedisStore) ScoredAppend(ctx context.Context, key string, value interface{}) error {
	highItem, err := s.Client.ZRevRange(key, 0, 0).Result()
	if err != nil {
		return sdk.WithStack(err)
	}
	if len(highItem) == 0 {
		return s.ScoredSetAdd(ctx, key, value, 1)
	}

	maxScore, err := s.Client.ZScore(key, highItem[0]).Result()
	if err != nil {
		return sdk.WithStack(err)
	}
	return s.ScoredSetAdd(ctx, key, value, maxScore+1)
}

func (s *RedisStore) ScoredSetAdd(ctx context.Context, key string, value interface{}, score float64) error {
	btes, err := json.Marshal(value)
	if err != nil {
		return sdk.WithStack(err)
	}

	if err := s.Client.ZAdd(key, redis.Z{
		Member: string(btes),
		Score:  score,
	}).Err(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

const (
	MIN float64 = math.MaxFloat64 * -1
	MAX float64 = math.MaxFloat64
)

func (s *RedisStore) ScoredSetScan(ctx context.Context, key string, from, to float64, dest interface{}) error {
	min := "-inf"
	if from != MIN {
		min = strconv.FormatFloat(from, 'E', -1, 64)
	}
	max := "+inf"
	if to != MAX {
		max = strconv.FormatFloat(to, 'E', -1, 64)
	}

	values, err := s.Client.ZRangeByScore(key, redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
	if err != nil {
		return fmt.Errorf("redis zrange error: %v", err)
	}

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("non-pointer %v", v.Type())
	}
	v = v.Elem()
	if v.Kind() != reflect.Slice {
		return errors.New("the interface is not a slice.")
	}

	typ := reflect.TypeOf(v.Interface())
	v.Set(reflect.MakeSlice(typ, len(values), len(values)))

	for i := 0; i < v.Len(); i++ {
		m := v.Index(i).Interface()
		if err := json.Unmarshal([]byte(values[i]), &m); err != nil {
			return sdk.WrapError(err, "redis> cannot unmarshal %s", values[i])
		}
		v.Index(i).Set(reflect.ValueOf(m))
	}

	return nil
}
