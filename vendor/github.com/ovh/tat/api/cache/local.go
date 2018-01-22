package cache

import (
	"time"

	"gopkg.in/redis.v4"
)

//LocalCache is a dummy struct
type LocalCache struct{}

func (c *LocalCache) Append(key, value string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) BLPop(timeout time.Duration, keys ...string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) BRPop(timeout time.Duration, keys ...string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) BRPopLPush(source, destination string, timeout time.Duration) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) Decr(key string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) DecrBy(key string, decrement int64) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) Del(keys ...string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) Exists(key string) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) Expire(key string, expiration time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) ExpireAt(key string, tm time.Time) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) FlushDb() *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
func (c *LocalCache) Get(key string) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) GetBit(key string, offset int64) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) GetRange(key string, start, end int64) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) GetSet(key string, value interface{}) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) HDel(key string, fields ...string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) HExists(key, field string) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) HGet(key, field string) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) HGetAll(key string) *redis.StringStringMapCmd {
	return redis.NewStringStringMapResult(map[string]string{}, nil)
}
func (c *LocalCache) HIncrBy(key, field string, incr int64) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) HKeys(key string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) HLen(key string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) HMGet(key string, fields ...string) *redis.SliceCmd {
	return redis.NewSliceResult([]interface{}{}, nil)
}
func (c *LocalCache) HMSet(key string, fields map[string]string) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
func (c *LocalCache) HScan(key string, cursor uint64, match string, count int64) redis.Scanner {
	return redis.Scanner{}
}
func (c *LocalCache) HSet(key, field, value string) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) HSetNX(key, field, value string) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) HVals(key string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) Incr(key string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) IncrBy(key string, value int64) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) Info(...string) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) LIndex(key string, index int64) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) LInsert(key, op string, pivot, value interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) LInsertAfter(key string, pivot, value interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) LInsertBefore(key string, pivot, value interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) LLen(key string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) LPop(key string) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) LPush(key string, values ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) LPushX(key string, value interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) LRange(key string, start, stop int64) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) LRem(key string, count int64, value interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) LSet(key string, index int64, value interface{}) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
func (c *LocalCache) LTrim(key string, start, stop int64) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
func (c *LocalCache) MGet(keys ...string) *redis.SliceCmd {
	return redis.NewSliceResult([]interface{}{}, nil)
}
func (c *LocalCache) MSet(pairs ...interface{}) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
func (c *LocalCache) MSetNX(pairs ...interface{}) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) PExpire(key string, expiration time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) PExpireAt(key string, tm time.Time) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) Ping() *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
func (c *LocalCache) PTTL(key string) *redis.DurationCmd {
	return redis.NewDurationResult(time.Second, nil)
}
func (c *LocalCache) Persist(key string) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) Pipeline() *redis.Pipeline {
	return nil
}
func (c *LocalCache) PubSubChannels(pattern string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) PubSubNumPat() *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) Publish(channel, message string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) RPop(key string) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) RPopLPush(source, destination string) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) RPush(key string, values ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) RPushX(key string, value interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) Rename(key, newkey string) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
func (c *LocalCache) RenameNX(key, newkey string) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) SAdd(key string, members ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) SCard(key string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) SDiff(keys ...string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) SDiffStore(destination string, keys ...string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) SInter(keys ...string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) SInterStore(destination string, keys ...string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) SIsMember(key string, member interface{}) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) SMembers(key string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) SMove(source, destination string, member interface{}) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) SPop(key string) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) SPopN(key string, count int64) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) SRandMember(key string) *redis.StringCmd {
	return redis.NewStringResult([]byte{}, nil)
}
func (c *LocalCache) SRandMemberN(key string, count int64) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) SRem(key string, members ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) SScan(key string, cursor uint64, match string, count int64) redis.Scanner {
	return redis.Scanner{}
}
func (c *LocalCache) SUnion(keys ...string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) SUnionStore(destination string, keys ...string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) Scan(cursor uint64, match string, count int64) redis.Scanner {
	return redis.Scanner{}
}
func (c *LocalCache) Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
func (c *LocalCache) SetBit(key string, offset int64, value int) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) SetRange(key string, offset int64, value string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) SetXX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}
func (c *LocalCache) Sort(key string, sort redis.Sort) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}
func (c *LocalCache) StrLen(key string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) TTL(key string) *redis.DurationCmd {
	return redis.NewDurationResult(time.Second, nil)
}
func (c *LocalCache) Type(key string) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
func (c *LocalCache) ZAdd(key string, members ...redis.Z) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) ZAddCh(key string, members ...redis.Z) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) ZAddNX(key string, members ...redis.Z) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) ZAddNXCh(key string, members ...redis.Z) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) ZAddXX(key string, members ...redis.Z) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) ZAddXXCh(key string, members ...redis.Z) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) ZCard(key string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) ZCount(key, min, max string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
func (c *LocalCache) ZIncr(key string, member redis.Z) *redis.FloatCmd {
	return redis.NewFloatResult(0.0, nil)
}
func (c *LocalCache) ZIncrBy(key string, increment float64, member string) *redis.FloatCmd {
	return redis.NewFloatResult(0.0, nil)
}
func (c *LocalCache) ZIncrNX(key string, member redis.Z) *redis.FloatCmd {
	return redis.NewFloatResult(0.0, nil)
}
func (c *LocalCache) ZIncrXX(key string, member redis.Z) *redis.FloatCmd {
	return redis.NewFloatResult(0.0, nil)
}
func (c *LocalCache) ZInterStore(destination string, store redis.ZStore, keys ...string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (c *LocalCache) ZRange(key string, start, stop int64) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}

func (c *LocalCache) ZRangeByLex(key string, opt redis.ZRangeBy) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}

func (c *LocalCache) ZRangeByScore(key string, opt redis.ZRangeBy) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}

func (c *LocalCache) ZRangeByScoreWithScores(key string, opt redis.ZRangeBy) *redis.ZSliceCmd {
	return redis.NewZSliceCmdResult([]redis.Z{}, nil)
}

func (c *LocalCache) ZRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd {
	return redis.NewZSliceCmdResult([]redis.Z{}, nil)
}

func (c *LocalCache) ZRank(key, member string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (c *LocalCache) ZRem(key string, members ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (c *LocalCache) ZRemRangeByRank(key string, start, stop int64) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (c *LocalCache) ZRemRangeByScore(key, min, max string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (c *LocalCache) ZRevRange(key string, start, stop int64) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}

func (c *LocalCache) ZRevRangeByLex(key string, opt redis.ZRangeBy) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}

func (c *LocalCache) ZRevRangeByScore(key string, opt redis.ZRangeBy) *redis.StringSliceCmd {
	return redis.NewStringSliceResult([]string{}, nil)
}

func (c *LocalCache) ZRevRangeByScoreWithScores(key string, opt redis.ZRangeBy) *redis.ZSliceCmd {
	return redis.NewZSliceCmdResult([]redis.Z{}, nil)
}

func (c *LocalCache) ZRevRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd {
	return redis.NewZSliceCmdResult([]redis.Z{}, nil)
}

func (c *LocalCache) ZRevRank(key, member string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (c *LocalCache) ZScan(key string, cursor uint64, match string, count int64) redis.Scanner {
	return redis.Scanner{}
}
func (c *LocalCache) ZScore(key, member string) *redis.FloatCmd {
	return redis.NewFloatResult(0.0, nil)
}
func (c *LocalCache) ZUnionStore(dest string, store redis.ZStore, keys ...string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}
