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
	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/engine/cdn/redis"
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
	Set(u sdk.CDNUnit)
	Init(ctx context.Context, cfg interface{}, goRoutines *sdk.GoRoutines) error
	ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error)
	Lock()
	Unlock()
	Status(ctx context.Context) []sdk.MonitoringStatusLine
}

type AbstractUnit struct {
	sync.Mutex
	u sdk.CDNUnit
}

func (a *AbstractUnit) ExistsInDatabase(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string) (*sdk.CDNItemUnit, error) {
	iu, err := LoadItemUnitByUnit(ctx, m, db, a.ID(), id, gorpmapper.GetOptions.WithDecryption)
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

func (a *AbstractUnit) Set(u sdk.CDNUnit) { a.u = u }

type BufferUnit interface {
	Interface
	Add(i sdk.CDNItemUnit, score uint, value string) error
	Append(i sdk.CDNItemUnit, value string) error
	Card(i sdk.CDNItemUnit) (int, error)
	NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error)
	NewAdvancedReader(ctx context.Context, i sdk.CDNItemUnit, format redis.ReaderFormat, from int64, to uint) (io.ReadCloser, error)
	Read(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
}

type StorageUnit interface {
	Interface
	NewWriter(ctx context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error)
	NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error)
	Write(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
	Read(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
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
	Local  *LocalStorageConfiguration  `toml:"local" json:"local,omitempty" mapstructure:"local"`
	Swift  *SwiftStorageConfiguration  `toml:"swift" json:"swift,omitempty" mapstructure:"swift"`
	Webdav *WebdavStorageConfiguration `toml:"webdav" json:"webdav,omitempty" mapstructure:"webdav"`
	CDS    *CDSStorageConfiguration    `toml:"cds" json:"cds,omitempty" mapstructure:"cds"`
}

type LocalStorageConfiguration struct {
	Path       string                                  `toml:"path" json:"path"`
	Encryption []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
}

type CDSStorageConfiguration struct {
	Host                  string                                  `toml:"host" json:"host"`
	InsecureSkipVerifyTLS bool                                    `toml:"insecureSkipVerifyTLS" json:"insecureSkipVerifyTLS"`
	Token                 string                                  `toml:"token" json:"-" comment:"consumer token must have the scopes Project (READ) and Run (READ)"`
	Encryption            []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
}

type SwiftStorageConfiguration struct {
	Address         string                                  `toml:"address" json:"address"`
	Username        string                                  `toml:"username" json:"username"`
	Password        string                                  `toml:"password" json:"-"`
	Tenant          string                                  `toml:"tenant" json:"tenant"`
	Domain          string                                  `toml:"domain" json:"domain"`
	Region          string                                  `toml:"region" json:"region"`
	ContainerPrefix string                                  `toml:"container_prefix" json:"container_prefix"`
	Encryption      []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
}

type WebdavStorageConfiguration struct {
	Address    string                                  `toml:"address" json:"address"`
	Username   string                                  `toml:"username" json:"username"`
	Password   string                                  `toml:"password" json:"password"`
	Path       string                                  `toml:"path" json:"path"`
	Encryption []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
}

type RedisBufferConfiguration struct {
	Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
	Password string `toml:"password" json:"-"`
}

type RunningStorageUnits struct {
	m        *gorpmapper.Mapper
	db       *gorp.DbMap
	config   Configuration
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

func (r *RunningStorageUnits) Start(ctx context.Context) error {
	scheduler := cron.New(cron.WithLocation(time.UTC), cron.WithSeconds())

	for i := range r.Storages {
		var cronSetting string
		for j := range r.config.Storages {
			if r.config.Storages[j].Name == r.Storages[i].Name() {
				cronSetting = r.config.Storages[j].Cron
				break
			}
		}
		if cronSetting == "" {
			return sdk.WithStack(fmt.Errorf("missing cron config for storage %s", r.Storages[i].Name()))
		}
		f := func(i int) error {
			_, err := scheduler.AddFunc(cronSetting, func() {
				if err := r.Run(ctx, r.Storages[i]); err != nil {
					log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				}
			})
			return sdk.WithStack(err)
		}
		if err := f(i); err != nil {
			return err
		}
	}

	scheduler.Start()

	go func() {
		<-ctx.Done()
		<-scheduler.Stop().Done()
	}()

	return nil
}

func Init(ctx context.Context, m *gorpmapper.Mapper, db *gorp.DbMap, config Configuration, goRoutines *sdk.GoRoutines) (*RunningStorageUnits, error) {
	var result = RunningStorageUnits{
		m:      m,
		db:     db,
		config: config,
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
	if err := bd.Init(ctx, config.Buffer.Redis, goRoutines); err != nil {
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
		_ = json.Unmarshal(b, &srvConfig) // nolint
		u = &sdk.CDNUnit{
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
	bd.Set(*u)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

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
			if err := sd.Init(ctx, cfg.CDS, goRoutines); err != nil {
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
			if err := sd.Init(ctx, cfg.Local, goRoutines); err != nil {
				return nil, err
			}
			storageUnit = sd
		case cfg.Swift != nil:
			d := GetDriver("swift")
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("swift driver is not a storage unit driver"))
			}
			if err := sd.Init(ctx, cfg.Swift, goRoutines); err != nil {
				return nil, err
			}
			storageUnit = sd
		case cfg.Webdav != nil:
			d := GetDriver("webdav")
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("webdav driver is not a storage unit driver"))
			}
			if err := sd.Init(ctx, cfg.Webdav, goRoutines); err != nil {
				return nil, err
			}
			storageUnit = sd
		default:
			return nil, sdk.WithStack(errors.New("unsupported storage unit"))
		}

		tx, err := db.Begin()
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		u, err := LoadUnitByName(ctx, m, tx, cfg.Name)
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			var srvConfig sdk.ServiceConfig
			b, _ := json.Marshal(cfg)
			_ = json.Unmarshal(b, &srvConfig) // nolint

			u = &sdk.CDNUnit{
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
		storageUnit.Set(*u)

		result.Storages = append(result.Storages, storageUnit)
		if err := tx.Commit(); err != nil {
			return nil, sdk.WithStack(err)
		}
	}

	return &result, nil
}

var (
	rs  = rand.NewSource(time.Now().Unix())
	rnd = rand.New(rs)
)

type Source interface {
	NewReader(context.Context) (io.ReadCloser, error)
	Read(io.Reader, io.Writer) error
	Name() string
}

type source interface {
	NewReader(context.Context, sdk.CDNItemUnit) (io.ReadCloser, error)
	Read(sdk.CDNItemUnit, io.Reader, io.Writer) error
	Name() string
}

type iuSource struct {
	iu     sdk.CDNItemUnit
	source source
}

func (s *iuSource) NewReader(ctx context.Context) (io.ReadCloser, error) {
	return s.source.NewReader(ctx, s.iu)
}
func (s *iuSource) Read(r io.Reader, w io.Writer) error {
	return s.source.Read(s.iu, r, w)
}
func (s *iuSource) Name() string {
	return s.source.Name()
}

func (r RunningStorageUnits) GetSource(ctx context.Context, i *sdk.CDNItem) (Source, error) {
	ok, err := r.Buffer.ItemExists(ctx, r.m, r.db, *i)
	if err != nil {
		return nil, err
	}

	if ok {
		iu, err := LoadItemUnitByUnit(ctx, r.m, r.db, r.Buffer.ID(), i.ID, gorpmapper.GetOptions.WithDecryption)
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
		return nil, sdk.WithStack(sdk.ErrNotFound)
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
