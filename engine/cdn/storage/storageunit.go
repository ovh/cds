package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	return drivers[typ]
}

var (
	drivers     = make(map[string]Interface)
	driversLock sync.Mutex
)

type Interface interface {
	Name() string
	Init(m *gorpmapper.Mapper, db *gorp.DbMap, u Unit, cfg interface{}) error
	ItemExists(i index.Item) error
}

type BufferUnit interface {
	Interface
	Add(i index.Item, score float64, value string) error
	Get(i index.Item, from, to float64) ([]string, error)
}

type StorageUnit interface {
	Interface
	Run()
	NewWriter(i index.Item) (io.WriteCloser, error)
	NewReader(i index.Item) (io.ReadCloser, error)
}

type Configuration struct {
	Buffer   BufferConfiguration    `toml:"buffer" json:"buffer"`
	Storages []StorageConfiguration `toml:"storages" json:"storages"`
}

type BufferConfiguration struct {
	Name  string                    `toml:"name" json:"name"`
	Redis *RedisBufferConfiguration `toml:"redis" json:"redis"`
}

type StorageConfiguration struct {
	Name     string                     `toml:"name" json:"name"`
	CronExpr string                     `toml:"cron" json:"cron"`
	Local    *LocalStorageConfiguration `toml:"local" json:"local"`
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

func Init(ctx context.Context, m *gorpmapper.Mapper, db *gorp.DbMap, config Configuration) (*RunningStorageUnits, error) {
	var result RunningStorageUnits

	// Start by initializing the buffer unit
	bufferUnit := Unit{
		Name: config.Buffer.Name,
	}
	switch {
	case config.Buffer.Redis != nil:
		d := GetDriver("redis")
		if d == nil {
			return nil, fmt.Errorf("redis driver is not available")
		}
		bd, is := d.(BufferUnit)
		if !is {
			return nil, fmt.Errorf("redis driver is not a buffer unit driver")
		}
		if err := bd.Init(m, db, bufferUnit, config.Buffer.Redis); err != nil {
			return nil, err
		}
		result.Buffer = bd

		u, err := LoadUnitByName(ctx, m, db, bd.Name())
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			var srvConfig sdk.ServiceConfig
			b, _ := json.Marshal(config.Buffer.Redis)
			json.Unmarshal(b, &srvConfig) // nolint

			u = &Unit{
				ID:      sdk.UUID(),
				Created: time.Now(),
				Name:    bd.Name(),
				Config:  srvConfig,
			}
			err = InsertUnit(ctx, m, db, u)
		}
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("no buffer unit has been configuration")
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
			if err := sd.Init(m, db, bufferUnit, cfg); err != nil {
				return nil, err
			}
			storageUnit = sd
		default:
			return nil, errors.New("unsupported storage unit")
		}

		schedulerEntry, err := scheduler.AddJob(cfg.CronExpr, storageUnit)
		if err != nil {
			return nil, err
		}
		result.Storages = append(result.Storages, storageUnit)
		log.Debug("cdn.storage.Init> storage scheduled: %v", schedulerEntry)
	}

	scheduler.Start()

	go func() {
		<-ctx.Done()
		<-scheduler.Stop().Done()
	}()

	return &result, nil
}
