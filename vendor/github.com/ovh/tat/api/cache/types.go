package cache

import (
	"time"

	"gopkg.in/redis.v4"
)

// Cache interface redis
type Cache interface {
	Append(key, value string) *redis.IntCmd
	BLPop(timeout time.Duration, keys ...string) *redis.StringSliceCmd
	BRPop(timeout time.Duration, keys ...string) *redis.StringSliceCmd
	BRPopLPush(source, destination string, timeout time.Duration) *redis.StringCmd
	Decr(key string) *redis.IntCmd
	DecrBy(key string, decrement int64) *redis.IntCmd
	Del(keys ...string) *redis.IntCmd
	Exists(key string) *redis.BoolCmd
	Expire(key string, expiration time.Duration) *redis.BoolCmd
	ExpireAt(key string, tm time.Time) *redis.BoolCmd
	FlushDb() *redis.StatusCmd
	Get(key string) *redis.StringCmd
	GetBit(key string, offset int64) *redis.IntCmd
	GetRange(key string, start, end int64) *redis.StringCmd
	GetSet(key string, value interface{}) *redis.StringCmd
	HDel(key string, fields ...string) *redis.IntCmd
	HExists(key, field string) *redis.BoolCmd
	HGet(key, field string) *redis.StringCmd
	HGetAll(key string) *redis.StringStringMapCmd
	HIncrBy(key, field string, incr int64) *redis.IntCmd
	HKeys(key string) *redis.StringSliceCmd
	HLen(key string) *redis.IntCmd
	HMGet(key string, fields ...string) *redis.SliceCmd
	HMSet(key string, fields map[string]string) *redis.StatusCmd
	HScan(key string, cursor uint64, match string, count int64) redis.Scanner
	HSet(key, field, value string) *redis.BoolCmd
	HSetNX(key, field, value string) *redis.BoolCmd
	HVals(key string) *redis.StringSliceCmd
	Incr(key string) *redis.IntCmd
	IncrBy(key string, value int64) *redis.IntCmd
	Info(section ...string) *redis.StringCmd
	LIndex(key string, index int64) *redis.StringCmd
	LInsert(key, op string, pivot, value interface{}) *redis.IntCmd
	LInsertAfter(key string, pivot, value interface{}) *redis.IntCmd
	LInsertBefore(key string, pivot, value interface{}) *redis.IntCmd
	LLen(key string) *redis.IntCmd
	LPop(key string) *redis.StringCmd
	LPush(key string, values ...interface{}) *redis.IntCmd
	LPushX(key string, value interface{}) *redis.IntCmd
	LRange(key string, start, stop int64) *redis.StringSliceCmd
	LRem(key string, count int64, value interface{}) *redis.IntCmd
	LSet(key string, index int64, value interface{}) *redis.StatusCmd
	LTrim(key string, start, stop int64) *redis.StatusCmd
	MGet(keys ...string) *redis.SliceCmd
	MSet(pairs ...interface{}) *redis.StatusCmd
	MSetNX(pairs ...interface{}) *redis.BoolCmd
	PExpire(key string, expiration time.Duration) *redis.BoolCmd
	PExpireAt(key string, tm time.Time) *redis.BoolCmd
	Ping() *redis.StatusCmd
	PTTL(key string) *redis.DurationCmd
	Persist(key string) *redis.BoolCmd
	Pipeline() *redis.Pipeline
	PubSubChannels(pattern string) *redis.StringSliceCmd
	PubSubNumPat() *redis.IntCmd
	Publish(channel, message string) *redis.IntCmd
	RPop(key string) *redis.StringCmd
	RPopLPush(source, destination string) *redis.StringCmd
	RPush(key string, values ...interface{}) *redis.IntCmd
	RPushX(key string, value interface{}) *redis.IntCmd
	Rename(key, newkey string) *redis.StatusCmd
	RenameNX(key, newkey string) *redis.BoolCmd
	SAdd(key string, members ...interface{}) *redis.IntCmd
	SCard(key string) *redis.IntCmd
	SDiff(keys ...string) *redis.StringSliceCmd
	SDiffStore(destination string, keys ...string) *redis.IntCmd
	SInter(keys ...string) *redis.StringSliceCmd
	SInterStore(destination string, keys ...string) *redis.IntCmd
	SIsMember(key string, member interface{}) *redis.BoolCmd
	SMembers(key string) *redis.StringSliceCmd
	SMove(source, destination string, member interface{}) *redis.BoolCmd
	SPop(key string) *redis.StringCmd
	SPopN(key string, count int64) *redis.StringSliceCmd
	SRandMember(key string) *redis.StringCmd
	SRandMemberN(key string, count int64) *redis.StringSliceCmd
	SRem(key string, members ...interface{}) *redis.IntCmd
	SScan(key string, cursor uint64, match string, count int64) redis.Scanner
	SUnion(keys ...string) *redis.StringSliceCmd
	SUnionStore(destination string, keys ...string) *redis.IntCmd
	Scan(cursor uint64, match string, count int64) redis.Scanner
	Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetBit(key string, offset int64, value int) *redis.IntCmd
	SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	SetRange(key string, offset int64, value string) *redis.IntCmd
	SetXX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Sort(key string, sort redis.Sort) *redis.StringSliceCmd
	StrLen(key string) *redis.IntCmd
	TTL(key string) *redis.DurationCmd
	Type(key string) *redis.StatusCmd
	ZAdd(key string, members ...redis.Z) *redis.IntCmd
	ZAddCh(key string, members ...redis.Z) *redis.IntCmd
	ZAddNX(key string, members ...redis.Z) *redis.IntCmd
	ZAddNXCh(key string, members ...redis.Z) *redis.IntCmd
	ZAddXX(key string, members ...redis.Z) *redis.IntCmd
	ZAddXXCh(key string, members ...redis.Z) *redis.IntCmd
	ZCard(key string) *redis.IntCmd
	ZCount(key, min, max string) *redis.IntCmd
	ZIncr(key string, member redis.Z) *redis.FloatCmd
	ZIncrBy(key string, increment float64, member string) *redis.FloatCmd
	ZIncrNX(key string, member redis.Z) *redis.FloatCmd
	ZIncrXX(key string, member redis.Z) *redis.FloatCmd
	ZInterStore(destination string, store redis.ZStore, keys ...string) *redis.IntCmd
	ZRange(key string, start, stop int64) *redis.StringSliceCmd
	ZRangeByLex(key string, opt redis.ZRangeBy) *redis.StringSliceCmd
	ZRangeByScore(key string, opt redis.ZRangeBy) *redis.StringSliceCmd
	ZRangeByScoreWithScores(key string, opt redis.ZRangeBy) *redis.ZSliceCmd
	ZRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd
	ZRank(key, member string) *redis.IntCmd
	ZRem(key string, members ...interface{}) *redis.IntCmd
	ZRemRangeByRank(key string, start, stop int64) *redis.IntCmd
	ZRemRangeByScore(key, min, max string) *redis.IntCmd
	ZRevRange(key string, start, stop int64) *redis.StringSliceCmd
	ZRevRangeByLex(key string, opt redis.ZRangeBy) *redis.StringSliceCmd
	ZRevRangeByScore(key string, opt redis.ZRangeBy) *redis.StringSliceCmd
	ZRevRangeByScoreWithScores(key string, opt redis.ZRangeBy) *redis.ZSliceCmd
	ZRevRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd
	ZRevRank(key, member string) *redis.IntCmd
	ZScan(key string, cursor uint64, match string, count int64) redis.Scanner
	ZScore(key, member string) *redis.FloatCmd
	ZUnionStore(dest string, store redis.ZStore, keys ...string) *redis.IntCmd
}
