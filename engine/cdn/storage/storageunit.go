package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/robfig/cron/v3"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func RegisterDriver(typ string, i Interface) {
	driversLock.Lock()
	defer driversLock.Unlock()
	drivers[typ] = i
}

func GetDriver(typ string) Interface {
	driversLock.Lock()
	defer driversLock.Unlock()
	ref := drivers[typ]
	i := reflect.ValueOf(ref).Elem()
	t := i.Type()
	v := reflect.New(t)
	return v.Interface().(Interface)
}

var (
	drivers     = make(map[string]Interface)
	driversLock sync.Mutex
)

type Interface interface {
	Name() string
	ID() string
	Set(m *gorpmapper.Mapper, db *gorp.DbMap, u Unit)
	GorpMapper() *gorpmapper.Mapper
	DB() *gorp.DbMap
	Init(cfg interface{}) error
	ItemExists(i index.Item) (bool, error)
}

type AbstractUnit struct {
	u  Unit
	m  *gorpmapper.Mapper
	db *gorp.DbMap
}

func (a *AbstractUnit) Name() string {
	return a.u.Name
}

func (a *AbstractUnit) ID() string {
	return a.u.ID
}

func (a *AbstractUnit) DB() *gorp.DbMap {
	return a.db
}

func (a *AbstractUnit) GorpMapper() *gorpmapper.Mapper {
	return a.m
}

func (a *AbstractUnit) Set(m *gorpmapper.Mapper, db *gorp.DbMap, u Unit) {
	a.u = u
	a.m = m
	a.db = db
}

type BufferUnit interface {
	Interface
	Add(i index.Item, score uint, value string) error
	Append(i index.Item, value string) error
	Get(i index.Item, from, to uint) ([]string, error)
	NewReader(i index.Item) (io.ReadCloser, error)
}

type StorageUnit interface {
	Interface
	NewWriter(i index.Item) (io.WriteCloser, error)
	NewReader(i index.Item) (io.ReadCloser, error)
}

type Configuration struct {
	Buffer   BufferConfiguration    `toml:"buffer" json:"buffer" mapstructure:"buffer"`
	Storages []StorageConfiguration `toml:"storages" json:"storages" mapstructure:"storages"`
}

type BufferConfiguration struct {
	Name  string                   `toml:"name" json:"name"`
	Redis RedisBufferConfiguration `toml:"redisBuffer" json:"redis" mapstructure:"redis"`
}

type StorageConfiguration struct {
	Name     string                     `toml:"name" json:"name"`
	CronExpr string                     `toml:"cron" json:"cron"`
	Local    *LocalStorageConfiguration `toml:"local" json:"local" mapstructure:"local"`
}

type LocalStorageConfiguration struct {
	Path string `toml:"path" json:"path"`
}

type RedisBufferConfiguration struct {
	Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
	Password string `toml:"password" json:"-"`
}

type RunningStorageUnits struct {
	Buffer   BufferUnit
	Storages []StorageUnit
}

func (r RunningStorageUnits) Storage(name string) StorageUnit {
	for _, x := range r.Storages {
		if x.Name() == name {
			return x
		}
	}
	return nil
}

func Init(ctx context.Context, m *gorpmapper.Mapper, db *gorp.DbMap, config Configuration) (*RunningStorageUnits, error) {
	var result RunningStorageUnits

	// Start by initializing the buffer unit
	d := GetDriver("redis")
	if d == nil {
		return nil, fmt.Errorf("redis driver is not available")
	}
	bd, is := d.(BufferUnit)
	if !is {
		return nil, fmt.Errorf("redis driver is not a buffer unit driver")
	}
	if err := bd.Init(config.Buffer.Redis); err != nil {
		return nil, err
	}
	result.Buffer = bd

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	u, err := LoadUnitByName(ctx, m, tx, config.Buffer.Name)
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		var srvConfig sdk.ServiceConfig
		b, _ := json.Marshal(config.Buffer.Redis)
		json.Unmarshal(b, &srvConfig) // nolint
		u = &Unit{
			ID:      sdk.UUID(),
			Created: time.Now(),
			Name:    config.Buffer.Name,
			Config:  srvConfig,
		}
		err = InsertUnit(ctx, m, tx, u)
	} else if err != nil {
		return nil, err
	}
	bd.Set(m, db, *u)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	scheduler := cron.New(cron.WithLocation(time.UTC), cron.WithSeconds())

	// Then initialize the storages unit
	for _, cfg := range config.Storages {
		var storageUnit StorageUnit
		switch {
		case cfg.Local != nil:
			d := GetDriver("local")
			if d == nil {
				return nil, fmt.Errorf("local driver is not available")
			}
			sd, is := d.(StorageUnit)
			if !is {
				return nil, fmt.Errorf("local driver is not a storage unit driver")
			}
			if err := sd.Init(cfg.Local); err != nil {
				return nil, err
			}
			storageUnit = sd
		default:
			return nil, errors.New("unsupported storage unit")
		}

		runFunc := func() {
			if err := result.Run(ctx, storageUnit); err != nil {
				log.Error(ctx, err.Error())
			}
		}

		schedulerEntry, err := scheduler.AddFunc(cfg.CronExpr, runFunc)
		if err != nil {
			return nil, err
		}

		tx, err := db.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()

		u, err := LoadUnitByName(ctx, m, tx, cfg.Name)
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			var srvConfig sdk.ServiceConfig
			b, _ := json.Marshal(cfg)
			json.Unmarshal(b, &srvConfig) // nolint

			u = &Unit{
				ID:      sdk.UUID(),
				Created: time.Now(),
				Name:    cfg.Name,
				Config:  srvConfig,
			}
			err = InsertUnit(ctx, m, tx, u)
		} else if err != nil {
			return nil, err
		}
		storageUnit.Set(m, db, *u)

		result.Storages = append(result.Storages, storageUnit)
		log.Debug("cdn.storage.Init> storage scheduled: %v", schedulerEntry)
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}

	scheduler.Start()

	go func() {
		<-ctx.Done()
		<-scheduler.Stop().Done()
	}()

	return &result, nil
}
