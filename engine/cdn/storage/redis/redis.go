package redis

import (
	"context"
	"fmt"
	"io"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

var (
	_         storage.LogBufferUnit = new(Redis)
	keyBuffer                       = cache.Key("cdn", "buffer")
)

type Redis struct {
	storage.AbstractUnit
	config     storage.RedisBufferConfiguration
	store      cache.ScoredSetStore
	bufferType storage.CDNBufferType
}

const driverName = "redis"

func init() {
	storage.RegisterDriver(driverName, new(Redis))
}

func (s *Redis) GetDriverName() string {
	return driverName
}

func (s *Redis) Init(_ context.Context, cfg interface{}, bufferType storage.CDNBufferType) error {
	config, is := cfg.(*storage.RedisBufferConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = *config
	var err error
	s.store, err = cache.New(sdk.RedisConf{Host: s.config.Host, Password: s.config.Password, DbIndex: s.config.DbIndex}, 60)
	if err != nil {
		return err
	}
	s.bufferType = bufferType
	return nil
}

func (s *Redis) BufferType() storage.CDNBufferType {
	return s.bufferType
}

func (s *Redis) Keys() ([]string, error) {
	return s.store.Keys(cache.Key(keyBuffer, "*"))
}

func (s *Redis) ItemExists(_ context.Context, _ *gorpmapper.Mapper, _ gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
	size, _ := s.store.SetCard(cache.Key(keyBuffer, i.ID))
	return size > 0, nil
}

func (s *Redis) Size(i sdk.CDNItemUnit) (int64, error) {
	k := cache.Key(keyBuffer, i.ItemID)
	return s.store.Size(k)
}

func (s *Redis) Add(i sdk.CDNItemUnit, score uint, since uint, value string) error {
	value = fmt.Sprintf("%d%d#%s", score, since, value)
	return s.store.ScoredSetAdd(context.Background(), cache.Key(keyBuffer, i.ItemID), value, float64(score))
}

func (s *Redis) Copy(ctx context.Context, srcItemID, destItemID string) error {
	res, err := s.store.ScoredSetScanWithScores(ctx, cache.Key(keyBuffer, srcItemID), cache.MIN, cache.MAX)
	if err != nil {
		return err
	}
	for _, r := range res {
		if err := s.store.ScoredSetAdd(ctx, cache.Key(keyBuffer, destItemID), r.Value, r.Score); err != nil {
			return err
		}
	}
	return nil
}

func (s *Redis) Card(i sdk.CDNItemUnit) (int, error) {
	return s.store.SetCard(cache.Key(keyBuffer, i.ItemID))
}

// NewReader instanciate a reader that it able to iterate over Redis storage unit
// with a score step of 100.0, starting at score 0
func (s *Redis) NewReader(_ context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	return &redis.Reader{
		Store:      s.store,
		PrefixKey:  keyBuffer,
		ItemID:     i.ItemID,
		ApiRefHash: i.Item.APIRefHash,
		Format:     sdk.CDNReaderFormatText,
	}, nil
}

// NewAdvancedReader instanciate a reader from given option, format can be JSON or Text. If from is < 0, read end lines (ex: from=-100 size=0 means read the last 100 lines)
func (s *Redis) NewAdvancedReader(_ context.Context, i sdk.CDNItemUnit, format sdk.CDNReaderFormat, from int64, size uint, sort int64) (io.ReadCloser, error) {
	return &redis.Reader{
		Store:      s.store,
		PrefixKey:  keyBuffer,
		ItemID:     i.ItemID,
		ApiRefHash: i.Item.APIRefHash,
		From:       from,
		Size:       size,
		Format:     format,
		Sort:       sort,
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
			Status:    sdk.MonitoringStatusOK,
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

func (s *Redis) ResyncWithDatabase(ctx context.Context, _ gorp.SqlExecutor, _ sdk.CDNItemType, _ bool) {
	log.Error(ctx, "Resynchronization with database not implemented for redis buffer unit")
}
