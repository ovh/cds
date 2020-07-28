package redis

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type Redis struct {
	name   string
	config storage.RedisBufferConfiguration
	store  cache.ScoredSetStore
}

var _ storage.BufferUnit = new(Redis)

func init() {
	storage.RegisterDriver("redis", new(Redis))
}

func (s *Redis) Name() string {
	return s.name
}
func (s *Redis) Init(m *gorpmapper.Mapper, db *gorp.DbMap, u storage.Unit, cfg interface{}) error {
	s.name = u.Name
	config, is := cfg.(*storage.RedisBufferConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = *config
	var err error
	s.store, err = cache.New(s.config.Host, s.config.Password, 60)
	return err
}

func (s *Redis) ItemExists(i index.Item) error {
	return nil
}

func (s *Redis) Add(i index.Item, index float64, value string) error {
	return nil
}

func (s *Redis) Get(i index.Item, from, to float64) ([]string, error) {
	return nil, nil
}
