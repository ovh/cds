package redis

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"

	"go.opencensus.io/stats"
)

var keyBuffer = cache.Key("cdn", "buffer")

type Redis struct {
	storage.AbstractUnit
	config storage.RedisBufferConfiguration
	store  cache.ScoredSetStore
}

var (
	_           storage.BufferUnit = new(Redis)
	metricsSize                    = stats.Int64("cdn/storage/redis/size", "redis size", stats.UnitDimensionless)
)

func init() {
	storage.RegisterDriver("redis", new(Redis))
}

func (s *Redis) Init(ctx context.Context, cfg interface{}) error {
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

	if err := telemetry.InitMetricsInt64(ctx, metricsSize); err != nil {
		return err
	}

	return nil
}

func (s *Redis) ItemExists(i sdk.CDNItem) (bool, error) {
	size, _ := s.store.SetCard(cache.Key(keyBuffer, i.ID))
	return size > 0, nil
}

func (s *Redis) Add(i sdk.CDNItemUnit, index uint, value string) error {
	value = strconv.Itoa(int(index)) + "#" + value
	return s.store.ScoredSetAdd(context.Background(), cache.Key(keyBuffer, i.ItemID), value, float64(index))
}

func (s *Redis) Append(i sdk.CDNItemUnit, value string) error {
	return s.store.ScoredSetAppend(context.Background(), cache.Key(keyBuffer, i.ItemID), value)
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
		From: 0,
		Size: 0,
	}, nil
}

func (s *Redis) Read(_ sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return err
}

func (s *Redis) Status(_ context.Context) []sdk.MonitoringStatusLine {
	if err := s.store.Ping(); err != nil {
		return []sdk.MonitoringStatusLine{{
			Component: "storage/redis/ping",
			Value:     "connect OK",
			Status:    sdk.MonitoringStatusAlert,
		}}
	}

	size, err := s.store.DBSize()
	if err != nil {
		return []sdk.MonitoringStatusLine{{
			Component: "storage/redis/size",
			Value:     fmt.Sprintf("ERROR while getting dbsize: %v", size),
			Status:    sdk.MonitoringStatusAlert,
		}}
	}

	return []sdk.MonitoringStatusLine{{
		Component: "storage/redis/size",
		Value:     fmt.Sprintf("%d keys", size),
		Status:    sdk.MonitoringStatusOK,
	}}
}
