package storage

import (
	"context"
	"io"
	"reflect"
	"sync"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/symmecrypt/convergent"
	"go.opencensus.io/stats"
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
	New(gorts *sdk.GoRoutines)
	Set(u sdk.CDNUnit)
	Init(ctx context.Context, cfg interface{}) error
	ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error)
	Lock()
	Unlock()
	Status(ctx context.Context) []sdk.MonitoringStatusLine
}

type AbstractUnit struct {
	sync.Mutex
	GoRoutines *sdk.GoRoutines
	u          sdk.CDNUnit
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

func (a *AbstractUnit) New(gorts *sdk.GoRoutines) { a.GoRoutines = gorts }

type BufferUnit interface {
	Interface
	Add(i sdk.CDNItemUnit, score uint, value string) error
	Append(i sdk.CDNItemUnit, value string) error
	Card(i sdk.CDNItemUnit) (int, error)
	NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error)
	NewAdvancedReader(ctx context.Context, i sdk.CDNItemUnit, format sdk.CDNReaderFormat, from int64, size uint) (io.ReadCloser, error)
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
	Name           string                      `toml:"name" json:"name"`
	Cron           string                      `toml:"cron" json:"cron" default:"*/30 * * * * ?"`
	CronItemNumber int64                       `toml:"cron_item_number" json:"cron_item_number" default:"100"`
	Local          *LocalStorageConfiguration  `toml:"local" json:"local,omitempty" mapstructure:"local"`
	Swift          *SwiftStorageConfiguration  `toml:"swift" json:"swift,omitempty" mapstructure:"swift"`
	Webdav         *WebdavStorageConfiguration `toml:"webdav" json:"webdav,omitempty" mapstructure:"webdav"`
	CDS            *CDSStorageConfiguration    `toml:"cds" json:"cds,omitempty" mapstructure:"cds"`
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
	Metrics  struct {
		StorageThroughput **stats.Float64Measure
	}
}
