package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// RedisStore a redis client and a default ttl
type RedisStore struct {
	ttl    int
	Client *redis.Client
}

type redisLogger struct{}

func (r redisLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	log.Debug(ctx, format, v...)
}

// NewRedisStore initiate a new redisStore
func NewRedisStore(redisConf sdk.RedisConf, ttl int) (*RedisStore, error) {
	var client *redis.Client

	var tlsConfig *tls.Config

	if redisConf.EnableTLS {
		tlsConfig = &tls.Config{InsecureSkipVerify: redisConf.InsecureSkipVerifyTLS, MinVersion: tls.VersionTLS13}
	}

	//if host is line master@localhost:26379,localhost:26380 => it's a redis sentinel cluster
	if strings.Contains(redisConf.Host, "@") {
		masterName := strings.Split(redisConf.Host, "@")[0]
		sentinelsStr := strings.Split(redisConf.Host, "@")[1]
		sentinels := strings.Split(sentinelsStr, ",")
		opts := &redis.FailoverOptions{
			MasterName:      masterName,
			SentinelAddrs:   sentinels,
			Password:        redisConf.Password,
			DB:              redisConf.DbIndex,
			PoolSize:        25,
			MaxRetries:      10,
			MinRetryBackoff: 30 * time.Millisecond,
			MaxRetryBackoff: 100 * time.Millisecond,
			TLSConfig:       tlsConfig,
		}
		client = redis.NewFailoverClient(opts)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:            redisConf.Host,
			Password:        redisConf.Password, // no password set
			DB:              redisConf.DbIndex,
			MaxRetries:      10,
			MinRetryBackoff: 30 * time.Millisecond,
			MaxRetryBackoff: 100 * time.Millisecond,
			TLSConfig:       tlsConfig,
		})
	}

	redis.SetLogger(new(redisLogger))

	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, sdk.WrapError(err, "unable to connect to redis %s:%d", redisConf.Host, redisConf.DbIndex)
	}
	if pong != "PONG" {
		return nil, fmt.Errorf("cannot ping Redis on %s", redisConf.Host)
	}
	return &RedisStore{
		ttl:    ttl,
		Client: client,
	}, nil
}

// DBSize: Return the number of keys in the currently-selected database
func (s *RedisStore) DBSize() (int64, error) {
	size, err := s.Client.DBSize(context.Background()).Result()
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return size, nil
}

func (s *RedisStore) Ping() error {
	pong, err := s.Client.Ping(context.Background()).Result()
	if err != nil {
		return sdk.WithStack(err)
	}
	if pong != "PONG" {
		return fmt.Errorf("cannot ping Redis")
	}
	return nil
}

func (s *RedisStore) Keys(pattern string) ([]string, error) {
	if s.Client == nil {
		return nil, sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}
	keys, err := s.Client.Keys(context.Background(), pattern).Result()
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

	val, errRedis := s.Client.Get(context.Background(), key).Result()
	if errRedis != nil && errRedis != redis.Nil {
		return false, sdk.WrapError(errRedis, "redis> get error %s", key)
	}
	if val != "" {
		if err := sdk.JSONUnmarshal([]byte(val), value); err != nil {
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

	if err := s.Client.Set(context.Background(), key, string(b), time.Duration(ttl)*time.Second).Err(); err != nil {
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

	if err := s.Client.Set(context.Background(), key, string(b), duration).Err(); err != nil {
		return sdk.WrapError(err, "set error %s", key)
	}

	return nil
}

// UpdateTTL update the ttl linked to the key
func (s *RedisStore) UpdateTTL(key string, ttl int) error {
	if s.Client == nil {
		return sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}

	if err := s.Client.Expire(context.Background(), key, time.Duration(ttl)*time.Second).Err(); err != nil {
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

	if err := s.Client.Del(context.Background(), key).Err(); err != nil {
		return sdk.WrapError(err, "redis> error deleting %s", key)
	}
	return nil
}

// DeleteAll delete all mathing keys in redis
func (s *RedisStore) DeleteAll(pattern string) error {
	if s.Client == nil {
		return sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}
	keys, err := s.Client.Keys(context.Background(), pattern).Result()
	if err != nil {
		return sdk.WrapError(err, "redis> Error deleting %s", pattern)
	}
	if len(keys) == 0 {
		return nil
	}
	if err := s.Client.Del(context.Background(), keys...).Err(); err != nil {
		return sdk.WrapError(err, "redis> Error deleting %s", pattern)
	}
	return nil
}

// Exist test is key exists
func (s *RedisStore) Exist(key string) (bool, error) {
	if s.Client == nil {
		return false, sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}
	ok, err := s.Client.Exists(context.Background(), key).Result()
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
	if err := s.Client.LPush(context.Background(), queueName, string(b)).Err(); err != nil {
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
	res, errRedis = s.Client.LLen(context.Background(), queueName).Result()
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
			res, err := s.Client.BRPop(context.Background(), time.Second, queueName).Result()
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
			return c.Err()
		}
	}
	if elem != "" {
		b := []byte(elem)
		if err := sdk.JSONUnmarshal(b, value); err != nil {
			return sdk.WrapError(err, "redis.DequeueWithContext> error on unmarshal value on queue:%s", queueName)
		}
	}
	return nil
}

// DequeueListWithContext gets from queue This is blocking while there is nothing in the queue, it can be cancelled with a context.Context
func (s *RedisStore) DequeueJSONRawMessagesWithContext(ctx context.Context, queueName string, waitDuration time.Duration, maxElements int) ([]json.RawMessage, error) {
	if s.Client == nil {
		return nil, sdk.WithStack(fmt.Errorf("redis> cannot get redis client"))
	}

	msgs := make([]json.RawMessage, 0, maxElements)
	ticker := time.NewTicker(waitDuration)
	defer ticker.Stop()
	for len(msgs) < maxElements {
		select {
		case <-ticker.C:
			if ctx.Err() != nil {
				return msgs, ctx.Err()
			}
			res, err := s.Client.BRPop(context.Background(), time.Second, queueName).Result()
			if err == redis.Nil {
				continue
			}
			if err == io.EOF {
				if len(msgs) > 0 {
					return msgs, nil
				}
				time.Sleep(1 * time.Second)
				continue
			}
			if err == nil {
				if len(res) == 0 {
					if len(msgs) > 0 {
						return msgs, nil
					}
				} else if len(res) == 2 {
					msgs = append(msgs, json.RawMessage(res[1]))
					continue
				}
			}
		case <-ctx.Done():
			return msgs, nil
		}
	}

	return msgs, nil
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
		_, errP := s.Client.Publish(context.Background(), channel, iUnquoted).Result()
		if errP == nil {
			break
		}
		log.Warn(ctx, "redis.Publish> Unable to publish in channel %s: %v", channel, errP)
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// Subscribe to a channel
func (s *RedisStore) Subscribe(channel string) (PubSub, error) {
	if s.Client == nil {
		return nil, fmt.Errorf("redis> cannot get redis client")
	}
	return &RedisPubSub{
		PubSub: s.Client.Subscribe(context.Background(), channel),
	}, nil
}

// RemoveFromQueue removes a member from a list
func (s *RedisStore) RemoveFromQueue(rootKey string, memberKey string) error {
	if err := s.Client.LRem(context.Background(), rootKey, 0, memberKey).Err(); err != nil {
		return sdk.WrapError(err, "error on RemoveFromQueue: rooKey:%v memberKey:%v", rootKey, memberKey)
	}
	return nil
}

// SetAdd add a member (identified by a key) in the cached set
func (s *RedisStore) SetAdd(rootKey string, memberKey string, member interface{}) error {
	return s.SetAddWithTTL(rootKey, memberKey, member, -1)
}

// SetAddSetAddWithTTL add a member (identified by a key) in the cached set
func (s *RedisStore) SetAddWithTTL(rootKey string, memberKey string, member interface{}, ttlInseconds int) error {
	err := s.Client.ZAdd(context.Background(), rootKey, redis.Z{
		Member: memberKey,
		Score:  float64(time.Now().UnixNano()),
	}).Err()
	if err != nil {
		return sdk.WrapError(err, "error on SetAdd")
	}
	return s.SetWithTTL(Key(rootKey, memberKey), member, ttlInseconds)
}

// SetRemove removes a member from a set
func (s *RedisStore) SetRemove(rootKey string, memberKey string, _ interface{}) error {
	if err := s.Client.ZRem(context.Background(), rootKey, memberKey).Err(); err != nil {
		return sdk.WrapError(err, "error on SetRemove")
	}
	return s.Delete(Key(rootKey, memberKey))
}

// SetCard returns the cardinality of a ZSet
func (s *RedisStore) SetCard(key string) (int, error) {
	v := s.Client.ZCard(context.Background(), key)
	return int(v.Val()), v.Err()
}

// SetScan scans a ZSet
func (s *RedisStore) SetScan(ctx context.Context, key string, members ...interface{}) error {
	values, err := s.Client.ZRangeByScore(context.Background(), key, &redis.ZRangeBy{
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
		res, err := s.Client.MGet(context.Background(), keys...).Result()
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
				if err := s.Client.ZRem(context.Background(), key, values[i]).Err(); err != nil {
					return sdk.WrapError(err, "redis>SetScan unable to delete member %s", keys[i])
				}
				log.Info(ctx, "redis> member %s deleted", keys[i])
				return sdk.WithStack(fmt.Errorf("SetScan member %s not found", keys[i]))
			}

			if err := sdk.JSONUnmarshal([]byte(res[i].(string)), members[i]); err != nil {
				return sdk.WrapError(err, "redis> cannot unmarshal %s", keys[i])
			}
		}
	}
	return nil
}

func (s *RedisStore) SetSearch(key, pattern string) ([]string, error) {
	keys, _, err := s.Client.ZScan(context.Background(), key, 0, pattern, 0).Result()
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
		res, errRedis = s.Client.SetNX(context.Background(), key, "true", expiration).Result()
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

func (s *RedisStore) Size(key string) (int64, error) {
	oct, err := s.Client.MemoryUsage(context.Background(), key).Result()
	return oct, sdk.WithStack(err)
}

func (s *RedisStore) ScoredSetGetScore(key string, member interface{}) (float64, error) {
	bts, err := json.Marshal(member)
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	score, err := s.Client.ZScore(context.Background(), key, string(bts)).Result()
	return score, sdk.WithStack(err)
}

func (s *RedisStore) ScoredSetAppend(ctx context.Context, key string, value interface{}) error {
	highItem, err := s.Client.ZRevRange(context.Background(), key, 0, 0).Result()
	if err != nil {
		return sdk.WithStack(err)
	}
	if len(highItem) == 0 {
		return s.ScoredSetAdd(ctx, key, value, 1)
	}

	maxScore, err := s.Client.ZScore(context.Background(), key, highItem[0]).Result()
	if err != nil {
		return sdk.WithStack(err)
	}
	return s.ScoredSetAdd(ctx, key, value, maxScore+1)
}

func (s *RedisStore) ScoredSetAdd(_ context.Context, key string, value interface{}, score float64) error {
	btes, err := json.Marshal(value)
	if err != nil {
		return sdk.WithStack(err)
	}

	if err := s.Client.ZAdd(context.Background(), key, redis.Z{
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

func (s *RedisStore) ScoredSetRem(_ context.Context, key string, members ...string) error {
	_, err := s.Client.ZRem(context.Background(), key, members).Result()
	return sdk.WithStack(err)
}

func (s *RedisStore) ScoredSetRange(_ context.Context, key string, from, to int64, dest interface{}) error {
	values, err := s.Client.ZRange(context.Background(), key, from, to).Result()
	if err != nil {
		return sdk.WithStack(fmt.Errorf("redis zrange error: %v", err))
	}

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return sdk.WithStack(fmt.Errorf("non-pointer %v", v.Type()))
	}
	v = v.Elem()
	if v.Kind() != reflect.Slice {
		return sdk.WithStack(errors.New("the interface is not a slice"))
	}

	typ := reflect.TypeOf(v.Interface())
	v.Set(reflect.MakeSlice(typ, len(values), len(values)))

	for i := 0; i < v.Len(); i++ {
		m := v.Index(i).Interface()
		if err := sdk.JSONUnmarshal([]byte(values[i]), &m); err != nil {
			return sdk.WrapError(err, "redis> cannot unmarshal %s", values[i])
		}
		v.Index(i).Set(reflect.ValueOf(m))
	}
	return nil
}

func (s *RedisStore) ScoredSetRevRange(_ context.Context, key string, offset int64, limit int64, dest interface{}) error {
	values, err := s.Client.ZRevRange(context.Background(), key, offset, limit).Result()
	if err != nil {
		return fmt.Errorf("redis zrevrange error: %v", err)
	}

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("non-pointer %v", v.Type())
	}
	v = v.Elem()
	if v.Kind() != reflect.Slice {
		return errors.New("the interface is not a slice")
	}

	typ := reflect.TypeOf(v.Interface())
	v.Set(reflect.MakeSlice(typ, len(values), len(values)))

	for i := 0; i < v.Len(); i++ {
		m := v.Index(i).Interface()
		if err := sdk.JSONUnmarshal([]byte(values[i]), &m); err != nil {
			return sdk.WrapError(err, "redis> cannot unmarshal %s", values[i])
		}
		v.Index(i).Set(reflect.ValueOf(m))
	}

	return nil
}

func (s *RedisStore) ScoredSetScan(_ context.Context, key string, from, to float64, dest interface{}) error {
	min := "-inf"
	if from != MIN {
		min = strconv.FormatFloat(from, 'E', -1, 64)
	}
	max := "+inf"
	if to != MAX {
		max = strconv.FormatFloat(to, 'E', -1, 64)
	}

	values, err := s.Client.ZRangeByScore(context.Background(), key, &redis.ZRangeBy{
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
		return errors.New("the interface is not a slice")
	}

	typ := reflect.TypeOf(v.Interface())
	v.Set(reflect.MakeSlice(typ, len(values), len(values)))

	for i := 0; i < v.Len(); i++ {
		m := v.Index(i).Interface()
		if err := sdk.JSONUnmarshal([]byte(values[i]), &m); err != nil {
			return sdk.WrapError(err, "redis> cannot unmarshal %s", values[i])
		}
		v.Index(i).Set(reflect.ValueOf(m))
	}

	return nil
}

func (s *RedisStore) ScoredSetScanWithScores(_ context.Context, key string, from, to float64) ([]SetValueWithScore, error) {
	min := "-inf"
	if from != MIN {
		min = strconv.FormatFloat(from, 'E', -1, 64)
	}
	max := "+inf"
	if to != MAX {
		max = strconv.FormatFloat(to, 'E', -1, 64)
	}

	values, err := s.Client.ZRangeByScoreWithScores(context.Background(), key, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
	if err != nil {
		return nil, sdk.WrapError(err, "redis zrange error")
	}

	res := make([]SetValueWithScore, len(values))
	for i := range values {
		res[i].Score = values[i].Score
		s, ok := values[i].Member.(string)
		if !ok {
			return nil, sdk.WithStack(fmt.Errorf("set value of type %T can't be cast to json.RawMessage", values[i].Member))
		}
		res[i].Value = json.RawMessage(s)
	}

	return res, nil
}

type RedisPubSub struct {
	*redis.PubSub
}

func (p *RedisPubSub) GetMessage(ctx context.Context) (string, error) {
	if msg, _ := p.PubSub.ReceiveTimeout(context.Background(), time.Second); msg != nil {
		if redisMsg, ok := msg.(*redis.Message); ok {
			return redisMsg.Payload, nil
		}
	}

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if msg, _ := p.PubSub.ReceiveTimeout(context.Background(), time.Second); msg != nil {
				if redisMsg, ok := msg.(*redis.Message); ok {
					return redisMsg.Payload, nil
				}
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func (s *RedisStore) ScoredSetScanMaxScore(_ context.Context, key string) (*SetValueWithScore, error) {
	values, err := s.Client.ZRevRangeByScoreWithScores(context.Background(), key, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  1,
	}).Result()
	if err != nil {
		return nil, sdk.WrapError(err, "redis zrange error")
	}

	if len(values) == 0 {
		return nil, nil
	}

	var res SetValueWithScore
	res.Score = values[0].Score
	rawValue, ok := values[0].Member.(string)
	if !ok {
		return nil, sdk.WithStack(fmt.Errorf("set value of type %T can't be cast to json.RawMessage", values[0].Member))
	}
	res.Value = json.RawMessage(rawValue)

	return &res, nil
}

func (s *RedisStore) Eval(expr string, args ...string) (string, error) {
	result, err := s.Client.Eval(context.Background(), expr, args).Result()
	if err != nil {
		return "", sdk.WithStack(err)
	}
	return fmt.Sprintf("%v", result), nil
}
