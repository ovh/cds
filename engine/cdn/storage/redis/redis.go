package redis

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
)

var keyBuffer = cache.Key("cdn", "buffer")

type Redis struct {
	storage.AbstractUnit
	config storage.RedisBufferConfiguration
	store  cache.ScoredSetStore
}

var _ storage.BufferUnit = new(Redis)

func init() {
	storage.RegisterDriver("redis", new(Redis))
}

func (s *Redis) Init(cfg interface{}) error {
	config, is := cfg.(storage.RedisBufferConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = config
	var err error
	s.store, err = cache.New(s.config.Host, s.config.Password, 60)
	return err
}

func (s *Redis) ItemExists(i index.Item) (bool, error) {
	size, _ := s.store.SetCard(cache.Key(keyBuffer, i.ID))
	return size > 0, nil
}

func (s *Redis) Add(i storage.ItemUnit, index uint, value string) error {
	value = strconv.Itoa(int(index)) + "#" + value
	return s.store.ScoredSetAdd(context.Background(), cache.Key(keyBuffer, i.ItemID), value, float64(index))
}

func (s *Redis) Append(i storage.ItemUnit, value string) error {
	return s.store.ScoredSetAppend(context.Background(), cache.Key(keyBuffer, i.ItemID), value)
}

func (s *Redis) Card(i storage.ItemUnit) (int, error) {
	return s.store.SetCard(cache.Key(keyBuffer, i.ItemID))
}

// NewReader instanciate a reader that it able to iterate over Redis storage unit
// with a score step of 100.0, starting at score 0
func (s *Redis) NewReader(i storage.ItemUnit) (io.ReadCloser, error) {
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

func (s *Redis) Read(i storage.ItemUnit, r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return err
}
