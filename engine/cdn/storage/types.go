package storage

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"math"
	"reflect"
	"sync"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/symmecrypt/convergent"
	"golang.org/x/crypto/pbkdf2"
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
	New(gorts *sdk.GoRoutines, syncParrallel int64, syncBandwidth float64)
	Set(u sdk.CDNUnit)
	ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error)
	Status(ctx context.Context) []sdk.MonitoringStatusLine
	SyncBandwidth() float64
	Remove(ctx context.Context, i sdk.CDNItemUnit) error
}

type AbstractUnit struct {
	GoRoutines    *sdk.GoRoutines
	u             sdk.CDNUnit
	syncChan      chan string
	syncBandwidth float64
}

func (a *AbstractUnit) ExistsInDatabase(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string) (*sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE unit_id = $1 and item_id = $2 LIMIT 1").Args(a.ID(), id)
	return getItemUnit(ctx, m, db, query, gorpmapper.GetOptions.WithDecryption)
}

func (a *AbstractUnit) Name() string {
	return a.u.Name
}

func (a *AbstractUnit) ID() string {
	return a.u.ID
}

func (a *AbstractUnit) Set(u sdk.CDNUnit) { a.u = u }

func (a *AbstractUnit) New(gorts *sdk.GoRoutines, syncParrallel int64, syncBandwidth float64) {
	a.GoRoutines = gorts
	a.syncChan = make(chan string, syncParrallel)
	if syncBandwidth <= 0 {
		syncBandwidth = math.MaxFloat64
	}
	a.syncBandwidth = syncBandwidth / float64(syncParrallel)
}

func (a *AbstractUnit) SyncItemChannel() chan string { return a.syncChan }

func (a *AbstractUnit) SyncBandwidth() float64 {
	return a.syncBandwidth
}

type BufferUnit interface {
	Interface
	Init(ctx context.Context, cfg interface{}, maxStepLog, maxServiceLog int64) error
	Add(i sdk.CDNItemUnit, score uint, value string, options WithOption) (int64, error)
	Card(i sdk.CDNItemUnit) (int, error)
	Size(i sdk.CDNItemUnit) (int64, error)
	NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error)
	NewAdvancedReader(ctx context.Context, i sdk.CDNItemUnit, format sdk.CDNReaderFormat, from int64, size uint, sort int64) (io.ReadCloser, error)
	Read(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
}

type StorageUnit interface {
	Interface
	Init(ctx context.Context, cfg interface{}) error
	SyncItemChannel() chan string
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
	HashLocatorSalt string                 `toml:"hashLocatorSalt" json:"hash_locator_salt" mapstructure:"hashLocatorSalt"`
	Buffer          BufferConfiguration    `toml:"buffer" json:"buffer" mapstructure:"buffer"`
	Storages        []StorageConfiguration `toml:"storages" json:"storages" mapstructure:"storages"`
}

type BufferConfiguration struct {
	Name  string                   `toml:"name" default:"redis" json:"name"`
	Redis RedisBufferConfiguration `toml:"redis" json:"redis" mapstructure:"redis"`
}

type StorageConfiguration struct {
	Name          string                      `toml:"name" json:"name"`
	SyncParallel  int64                       `toml:"syncParallel" json:"sync_parallel" comment:"number of parallel sync processes"`
	SyncBandwidth int64                       `toml:"syncBandwidth" json:"sync_bandwidth" comment:"global bandwith shared by the sync processes (in Mb)"`
	Local         *LocalStorageConfiguration  `toml:"local" json:"local,omitempty" mapstructure:"local"`
	Swift         *SwiftStorageConfiguration  `toml:"swift" json:"swift,omitempty" mapstructure:"swift"`
	Webdav        *WebdavStorageConfiguration `toml:"webdav" json:"webdav,omitempty" mapstructure:"webdav"`
	CDS           *CDSStorageConfiguration    `toml:"cds" json:"cds,omitempty" mapstructure:"cds"`
}

type LocalStorageConfiguration struct {
	Path       string                                  `toml:"path" json:"path"`
	Encryption []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
}

type CDSStorageConfiguration struct {
	Host                  string `toml:"host" json:"host"`
	InsecureSkipVerifyTLS bool   `toml:"insecureSkipVerifyTLS" json:"insecureSkipVerifyTLS"`
	Token                 string `toml:"token" json:"-" comment:"consumer token must have the scopes Project (READ) and Run (READ)"`
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

func (x RunningStorageUnits) HashLocator(loc string) string {
	salt := []byte(x.config.HashLocatorSalt)
	hashLocator := hex.EncodeToString(pbkdf2.Key([]byte(loc), salt, 4096, 32, sha1.New))
	return hashLocator
}

type LogConfig struct {
	// Step logs
	StepMaxSize         int64 `toml:"stepMaxSize" default:"15728640" comment:"Max step logs size in bytes (default: 15MB)" json:"stepMaxSize"`
	StepLinesRateLimit  int64 `toml:"stepLinesRateLimit" default:"1800" comment:"Number of lines that a worker can send by seconds" json:"stepLinesRateLimit"`
	NbJobLogsGoroutines int64 `toml:"nbJobLogsGoroutines" default:"5" comment:"Number of workers that dequeue the job log queue" json:"nbJobLogsGoroutines"`

	// Service logs
	ServiceMaxSize          int64 `toml:"serviceMaxSize" default:"15728640" comment:"Max service logs size in bytes (default: 15MB)" json:"serviceMaxSize"`
	NbServiceLogsGoroutines int64 `toml:"nbServiceLogsGoroutines" default:"5" comment:"Number of workers that dequeue the service log queue" json:"nbServiceLogsGoroutines"`
}

type WithOption struct {
	IslastLine bool
}
