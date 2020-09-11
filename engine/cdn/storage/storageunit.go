package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/symmecrypt/convergent"
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
	Lock()
	Unlock()
}

type AbstractUnit struct {
	sync.Mutex
	u  Unit
	m  *gorpmapper.Mapper
	db *gorp.DbMap
}

func (a *AbstractUnit) ExistsInDatabase(id string) (*ItemUnit, error) {
	iu, err := LoadItemUnitByUnit(context.Background(), a.GorpMapper(), a.DB(), a.ID(), id, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return nil, err
	}
	return iu, nil
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
	Add(i ItemUnit, score uint, value string) error
	Append(i ItemUnit, value string) error
	Card(i ItemUnit) (int, error)
	NewReader(i ItemUnit) (io.ReadCloser, error)
	Read(i ItemUnit, r io.Reader, w io.Writer) error
}

type StorageUnit interface {
	Interface
	NewWriter(i ItemUnit) (io.WriteCloser, error)
	NewReader(i ItemUnit) (io.ReadCloser, error)
	Write(i ItemUnit, r io.Reader, w io.Writer) error
	Read(i ItemUnit, r io.Reader, w io.Writer) error
}

type StorageUnitWithLocator interface {
	StorageUnit
	NewLocator(s string) (string, error)
}

type Configuration struct {
	Buffer   BufferConfiguration    `toml:"buffer" json:"buffer" mapstructure:"buffer"`
	Storages []StorageConfiguration `toml:"storages" json:"storages" mapstructure:"storages"`
}

type BufferConfiguration struct {
	Name  string                   `toml:"name" default:"redis" json:"name"`
	Redis RedisBufferConfiguration `toml:"redis" json:"redis" mapstructure:"redis"`
}

type StorageConfiguration struct {
	Name   string                      `toml:"name" json:"name"`
	Cron   string                      `toml:"cron" json:"cron"`
	Local  *LocalStorageConfiguration  `toml:"local" json:"local" mapstructure:"local"`
	Swift  *SwiftStorageConfiguration  `toml:"swift" json:"swift" mapstructure:"swift"`
	Webdav *WebdavStorageConfiguration `toml:"webdav" json:"webdav" mapstructure:"webdav"`
	CDS    *CDSStorageConfiguration    `toml:"cds" json:"cds" mapstructure:"cds"`
}

type LocalStorageConfiguration struct {
	Path       string                                  `toml:"path" json:"path"`
	Encryption []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"encryption" mapstructure:"encryption"`
}

type CDSStorageConfiguration struct {
	Host                  string                                  `toml:"host" json:"host"`
	InsecureSkipVerifyTLS bool                                    `toml:"insecureSkipVerifyTLS" json:"insecureSkipVerifyTLS"`
	Token                 string                                  `toml:"token" json:"token"`
	Encryption            []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"encryption" mapstructure:"encryption"`
}

type SwiftStorageConfiguration struct {
	Address         string                                  `toml:"address" json:"address"`
	Username        string                                  `toml:"username" json:"username"`
	Password        string                                  `toml:"password" json:"password"`
	Tenant          string                                  `toml:"tenant" json:"tenant"`
	Domain          string                                  `toml:"domain" json:"domain"`
	Region          string                                  `toml:"region" json:"region"`
	ContainerPrefix string                                  `toml:"container_prefix" json:"container_prefix"`
	Encryption      []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"encryption" mapstructure:"encryption"`
}

type WebdavStorageConfiguration struct {
	Address    string                                  `toml:"address" json:"address"`
	Username   string                                  `toml:"username" json:"username"`
	Password   string                                  `toml:"password" json:"password"`
	Path       string                                  `toml:"path" json:"path"`
	Encryption []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"encryption" mapstructure:"encryption"`
}

type RedisBufferConfiguration struct {
	Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
	Password string `toml:"password" json:"-"`
}

type RunningStorageUnits struct {
	m        *gorpmapper.Mapper
	db       *gorp.DbMap
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
	var result = RunningStorageUnits{
		m:  m,
		db: db,
	}

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
	defer tx.Rollback() // nolint

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
		if err := InsertUnit(ctx, m, tx, u); err != nil {
			return nil, err
		}
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
		case cfg.CDS != nil:
			d := GetDriver("cds")
			if d == nil {
				return nil, sdk.WithStack(fmt.Errorf("cds driver is not available"))
			}
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("cds driver is not a storage unit driver"))
			}
			if err := sd.Init(cfg.CDS); err != nil {
				return nil, err
			}
			storageUnit = sd
		case cfg.Local != nil:
			d := GetDriver("local")
			if d == nil {
				return nil, sdk.WithStack(fmt.Errorf("local driver is not available"))
			}
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("local driver is not a storage unit driver"))
			}
			if err := sd.Init(cfg.Local); err != nil {
				return nil, err
			}
			storageUnit = sd
		case cfg.Swift != nil:
			d := GetDriver("swift")
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("swift driver is not a storage unit driver"))
			}
			if err := sd.Init(cfg.Swift); err != nil {
				return nil, err
			}
			storageUnit = sd
		case cfg.Webdav != nil:
			d := GetDriver("webdav")
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("webdav driver is not a storage unit driver"))
			}
			if err := sd.Init(cfg.Webdav); err != nil {
				return nil, err
			}
			storageUnit = sd
		default:
			return nil, sdk.WithStack(errors.New("unsupported storage unit"))
		}

		runFunc := func() {
			if err := result.Run(ctx, storageUnit); err != nil {
				log.Error(ctx, "cdn:storageunit: %v", err.Error())
			}
		}

		if _, err := scheduler.AddFunc(cfg.Cron, runFunc); err != nil {
			return nil, sdk.WithStack(err)
		}

		tx, err := db.Begin()
		if err != nil {
			return nil, sdk.WithStack(err)
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
		}
		if err != nil {
			return nil, err
		}
		storageUnit.Set(m, db, *u)

		result.Storages = append(result.Storages, storageUnit)
		if err := tx.Commit(); err != nil {
			return nil, sdk.WithStack(err)
		}
	}

	scheduler.Start()

	go func() {
		<-ctx.Done()
		<-scheduler.Stop().Done()
	}()

	return &result, nil
}

var (
	rs  = rand.NewSource(time.Now().Unix())
	rnd = rand.New(rs)
)

type Source interface {
	NewReader() (io.ReadCloser, error)
	Read(io.Reader, io.Writer) error
	Name() string
}

type source interface {
	NewReader(ItemUnit) (io.ReadCloser, error)
	Read(ItemUnit, io.Reader, io.Writer) error
	Name() string
}

type iuSource struct {
	iu     ItemUnit
	source source
}

func (s *iuSource) NewReader() (io.ReadCloser, error) {
	return s.source.NewReader(s.iu)
}
func (s *iuSource) Read(r io.Reader, w io.Writer) error {
	return s.source.Read(s.iu, r, w)
}
func (s *iuSource) Name() string {
	return s.source.Name()
}

func (r RunningStorageUnits) GetSource(ctx context.Context, i *index.Item) (Source, error) {
	ok, err := r.Buffer.ItemExists(*i)
	if err != nil {
		return nil, err
	}

	if ok {
		iu, err := LoadItemUnitByUnit(ctx, r.Buffer.GorpMapper(), r.Buffer.DB(), r.Buffer.ID(), i.ID, gorpmapper.GetOptions.WithDecryption)
		if err != nil {
			return nil, err
		}
		return &iuSource{iu: *iu, source: r.Buffer}, nil
	}

	// Find a storage unit where the item is complete
	itemUnits, err := LoadAllItemUnitsByItemID(ctx, r.m, r.db, i.ID)
	if err != nil {
		return nil, err
	}

	if len(itemUnits) == 0 {
		log.Warning(ctx, "item %s can't be found. No unit knows it...", i.ID)
		return nil, sdk.ErrNotFound
	}

	// Random pick a unit
	idx := 0
	if len(itemUnits) > 1 {
		idx = rnd.Intn(len(itemUnits))
	}
	refItemUnit := itemUnits[idx]
	refUnitID := refItemUnit.UnitID
	refUnit, err := LoadUnitByID(ctx, r.m, r.db, refUnitID)
	if err != nil {
		return nil, err
	}

	unit := r.Storage(refUnit.Name)
	if unit == nil {
		return nil, sdk.WithStack(fmt.Errorf("unable to find unit %s", refUnit.Name))
	}

	return &iuSource{iu: *&refItemUnit, source: unit}, nil
}
