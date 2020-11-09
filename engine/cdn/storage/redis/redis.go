package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

var (
	_                storage.BufferUnit = new(Redis)
	keyBuffer                           = cache.Key("cdn", "buffer")
	luaAddLogLineExp                    = `
		local size = redis.call('MEMORY', 'USAGE', KEYS[1]);
		if not(size) or size < tonumber(KEYS[4]) then
			redis.call('zadd', KEYS[1], tonumber(KEYS[3]), KEYS[2]);

			size = redis.call('MEMORY', 'USAGE', KEYS[1]);
			if size > tonumber(KEYS[4]) and KEYS[5] ~= 'true' then
				redis.call('zadd', KEYS[1], tonumber(KEYS[3]), '\"' .. tonumber(KEYS[3])+1 .. '#...truncated\\n\"');
			end
		end
		return size;
	`
)

type Redis struct {
	storage.AbstractUnit
	config            storage.RedisBufferConfiguration
	store             cache.ScoredSetStore
	maxStepLogSize    int64
	maxServiceLogSize int64
}

func init() {
	storage.RegisterDriver("redis", new(Redis))
}

func (s *Redis) Init(_ context.Context, cfg interface{}, maxStepLog, maxServiceLog int64) error {
	s.maxStepLogSize = maxStepLog
	s.maxServiceLogSize = maxServiceLog
	config, is := cfg.(storage.RedisBufferConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = config
	var err error
	s.store, err = cache.New(s.config.Host, s.config.Password, 60)
	if err != nil {
		return err
	}

	return nil
}

func (s *Redis) ItemExists(_ context.Context, _ *gorpmapper.Mapper, _ gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
	size, _ := s.store.SetCard(cache.Key(keyBuffer, i.ID))
	return size > 0, nil
}

func (s *Redis) Size(i sdk.CDNItemUnit) (int64, error) {
	k := cache.Key(keyBuffer, i.ItemID)
	return s.store.Size(k)
}

func (s *Redis) Add(i sdk.CDNItemUnit, index uint, value string, options storage.WithOption) (int64, error) {
	var maxsize int64
	switch i.Item.Type {
	case sdk.CDNTypeItemServiceLog:
		maxsize = s.maxServiceLogSize
	default:
		maxsize = s.maxStepLogSize
	}

	value = strconv.Itoa(int(index)) + "#" + value

	btes, err := json.Marshal(value)
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	result, err := s.store.Eval(luaAddLogLineExp,
		cache.Key(keyBuffer, i.ItemID),
		string(btes),
		strconv.FormatUint(uint64(index), 10),
		strconv.Itoa(int(maxsize)),
		strconv.FormatBool(options.IslastLine),
	)
	if err != nil {
		return 0, err
	}

	resultInt, err := strconv.Atoi(result)
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	return int64(resultInt), nil
}

func (s *Redis) Card(i sdk.CDNItemUnit) (int, error) {
	return s.store.SetCard(cache.Key(keyBuffer, i.ItemID))
}

// NewReader instanciate a reader that it able to iterate over Redis storage unit
// with a score step of 100.0, starting at score 0
func (s *Redis) NewReader(_ context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	return &redis.Reader{
		ReadWrite: redis.ReadWrite{
			Store:     s.store,
			PrefixKey: keyBuffer,
			ItemID:    i.ItemID,
			UsageKey:  "",
		},
		Format: sdk.CDNReaderFormatText,
	}, nil
}

// NewAdvancedReader instanciate a reader from given option, format can be JSON or Text. If from is < 0, read end lines (ex: from=-100 size=0 means read the last 100 lines)
func (s *Redis) NewAdvancedReader(_ context.Context, i sdk.CDNItemUnit, format sdk.CDNReaderFormat, from int64, size uint, sort int64) (io.ReadCloser, error) {
	return &redis.Reader{
		ReadWrite: redis.ReadWrite{
			Store:     s.store,
			PrefixKey: keyBuffer,
			ItemID:    i.ItemID,
			UsageKey:  "",
		},
		From:   from,
		Size:   size,
		Format: format,
		Sort:   sort,
	}, nil
}

func (s *Redis) Read(_ sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return sdk.WithStack(err)
}

func (s *Redis) Status(_ context.Context) []sdk.MonitoringStatusLine {
	if err := s.store.Ping(); err != nil {
		return []sdk.MonitoringStatusLine{{
			Component: fmt.Sprintf("storage/%s/ping", s.Name()),
			Value:     "connect KO",
			Status:    sdk.MonitoringStatusAlert,
		}}
	}

	size, err := s.store.DBSize()
	if err != nil {
		return []sdk.MonitoringStatusLine{{
			Component: fmt.Sprintf("storage/%s/size", s.Name()),
			Value:     fmt.Sprintf("ERROR while getting dbsize: %v", size),
			Status:    sdk.MonitoringStatusAlert,
		}}
	}

	return []sdk.MonitoringStatusLine{
		{
			Component: fmt.Sprintf("storage/%s/ping", s.Name()),
			Value:     "connect OK",
			Status:    sdk.MonitoringStatusAlert,
		},
		{
			Component: fmt.Sprintf("storage/%s/redis_dbsize", s.Name()),
			Value:     fmt.Sprintf("%d keys", size),
			Status:    sdk.MonitoringStatusOK,
		}}
}

func (s *Redis) Remove(_ context.Context, i sdk.CDNItemUnit) error {
	return sdk.WithStack(s.store.Delete(cache.Key(keyBuffer, i.ItemID)))
}
