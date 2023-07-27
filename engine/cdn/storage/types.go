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
	"github.com/ovh/symmecrypt/convergent"
	"github.com/ovh/symmecrypt/keyloader"
	"golang.org/x/crypto/pbkdf2"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
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
	New(gorts *sdk.GoRoutines, config AbstractUnitConfig)
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
	disableSync   bool
}

func (a *AbstractUnit) ExistsInDatabase(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, id string) (*sdk.CDNItemUnit, error) {
	query := gorpmapper.NewQuery("SELECT * FROM storage_unit_item WHERE unit_id = $1 and item_id = $2 LIMIT 1").Args(a.ID(), id)
	return getItemUnit(ctx, m, db, query, gorpmapper.GetOptions.WithDecryption)
}

func (a *AbstractUnit) CanSync() bool {
	return !a.disableSync
}

func (a *AbstractUnit) Name() string {
	return a.u.Name
}

func (a *AbstractUnit) ID() string {
	return a.u.ID
}

func (a *AbstractUnit) Set(u sdk.CDNUnit) { a.u = u }

func (a *AbstractUnit) New(gorts *sdk.GoRoutines, config AbstractUnitConfig) {
	a.GoRoutines = gorts
	a.syncChan = make(chan string, config.syncParrallel)
	if config.syncBandwidth <= 0 {
		config.syncBandwidth = math.MaxFloat64
	}
	a.syncBandwidth = config.syncBandwidth / float64(config.syncParrallel)
	a.disableSync = config.disableSync
}

func (a *AbstractUnit) SyncItemChannel() chan string { return a.syncChan }

func (a *AbstractUnit) SyncBandwidth() float64 {
	return a.syncBandwidth
}

type Unit interface {
	Read(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
	NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error)
	GetDriverName() string
	ResyncWithDatabase(ctx context.Context, db gorp.SqlExecutor, t sdk.CDNItemType, dryRun bool)
}

type BufferUnit interface {
	Interface
	Unit
	Init(ctx context.Context, cfg interface{}, bufferType CDNBufferType) error
	Size(i sdk.CDNItemUnit) (int64, error)
	Read(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
	BufferType() CDNBufferType
}

type LogBufferUnit interface {
	BufferUnit
	Add(i sdk.CDNItemUnit, score uint, since uint, value string) error
	Card(i sdk.CDNItemUnit) (int, error)
	NewAdvancedReader(ctx context.Context, i sdk.CDNItemUnit, format sdk.CDNReaderFormat, from int64, size uint, sort int64) (io.ReadCloser, error)
	Keys() ([]string, error)
}

type FileBufferUnit interface {
	BufferUnit
	NewWriter(ctx context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error)
	Write(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
}

type AbstractUnitConfig struct {
	syncParrallel int64
	syncBandwidth float64
	disableSync   bool
}

type StorageUnit interface {
	Interface
	Unit
	Init(ctx context.Context, cfg interface{}) error
	SyncItemChannel() chan string
	NewWriter(ctx context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error)
	Write(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error
	CanSync() bool
}

type StorageUnitWithLocator interface {
	StorageUnit
	NewLocator(s string) (string, error)
}

type Configuration struct {
	HashLocatorSalt string                          `toml:"hashLocatorSalt" json:"hash_locator_salt" mapstructure:"hashLocatorSalt"`
	Buffers         map[string]BufferConfiguration  `toml:"buffers" json:"buffers" mapstructure:"buffers"`
	Storages        map[string]StorageConfiguration `toml:"storages" json:"storages" mapstructure:"storages"`
	SyncSeconds     int                             `toml:"syncSeconds" default:"30" json:"syncSeconds" comment:"each n seconds, all storage backends will have to start a synchronization with the buffer"`
	SyncNbElements  int64                           `toml:"syncNbElements" default:"100" json:"syncNbElements" comment:"nb items to synchronize from the buffer"`
	PurgeSeconds    int                             `toml:"purgeSeconds" default:"5" json:"purgeSeconds" comment:"each n seconds, all storage backends will have to start to delete storage unit item with deleted flag"`
	PurgeNbElements int                             `toml:"purgeNbElements" default:"1000" json:"purgeNbElements" comment:"nb items to delete in each purge loop"`
}

type BufferConfiguration struct {
	Redis      *RedisBufferConfiguration `toml:"redis" json:"redis" mapstructure:"redis"`
	Local      *LocalBufferConfiguration `toml:"local" json:"local" mapstructure:"local"`
	Nfs        *NFSBufferConfiguration   `toml:"nfs" json:"nfs,omitempty" mapstructure:"nfs"`
	BufferType CDNBufferType             `toml:"bufferType" json:"bufferType" comment:"it can be 'log' to receive logs or 'file' to receive artifacts"`
}

type CDNBufferType string

const (
	CDNBufferTypeLog  CDNBufferType = "log"
	CDNBufferTypeFile CDNBufferType = "file"
)

type StorageConfiguration struct {
	SyncParallel  int64                       `toml:"syncParallel" json:"sync_parallel" comment:"number of parallel sync processes"`
	SyncBandwidth int64                       `toml:"syncBandwidth" json:"sync_bandwidth" comment:"global bandwith shared by the sync processes (in Mb)"`
	DisableSync   bool                        `toml:"disableSync" json:"disable_sync" comment:"flag to disabled backend synchronization"`
	Local         *LocalStorageConfiguration  `toml:"local" json:"local,omitempty" mapstructure:"local"`
	Swift         *SwiftStorageConfiguration  `toml:"swift" json:"swift,omitempty" mapstructure:"swift"`
	Webdav        *WebdavStorageConfiguration `toml:"webdav" json:"webdav,omitempty" mapstructure:"webdav"`
	S3            *S3StorageConfiguration     `toml:"s3" json:"s3,omitempty" mapstructure:"s3"`
}

type LocalStorageConfiguration struct {
	Path       string                                  `toml:"path" json:"path"`
	Encryption []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
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

type S3StorageConfiguration struct {
	BucketName          string                                  `toml:"bucketName" json:"bucketName" comment:"Name of the S3 bucket to use when storing artifacts"`
	Region              string                                  `toml:"region" json:"region" default:"us-east-1" comment:"The AWS region"`
	Prefix              string                                  `toml:"prefix" json:"prefix" comment:"A subfolder of the bucket to store objects in, if left empty will store at the root of the bucket"`
	AuthFromEnvironment bool                                    `toml:"authFromEnv" json:"authFromEnv" default:"false" comment:"Pull S3 auth information from env vars AWS_SECRET_ACCESS_KEY and AWS_SECRET_KEY_ID"`
	SharedCredsFile     string                                  `toml:"sharedCredsFile" json:"sharedCredsFile" comment:"The path for the AWS credential file, used with profile"`
	Profile             string                                  `toml:"profile" json:"profile" comment:"The profile within the AWS credentials file to use"`
	AccessKeyID         string                                  `toml:"accessKeyId" json:"accessKeyId" comment:"A static AWS Secret Key ID"`
	SecretAccessKey     string                                  `toml:"secretAccessKey" json:"-" comment:"A static AWS Secret Access Key"`
	SessionToken        string                                  `toml:"sessionToken" json:"-" comment:"A static AWS session token"`
	Endpoint            string                                  `toml:"endpoint" json:"endpoint" comment:"S3 API Endpoint (optional)" commented:"true"` //optional
	DisableSSL          bool                                    `toml:"disableSSL" json:"disableSSL" commented:"true"`                                  //optional
	ForcePathStyle      bool                                    `toml:"forcePathStyle" json:"forcePathStyle" commented:"true"`                          //optional
	Encryption          []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
}

type WebdavStorageConfiguration struct {
	Address    string                                  `toml:"address" json:"address"`
	Username   string                                  `toml:"username" json:"username"`
	Password   string                                  `toml:"password" json:"password"`
	Path       string                                  `toml:"path" json:"path"`
	Encryption []convergent.ConvergentEncryptionConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
}

type RedisBufferConfiguration struct {
	Host     string `toml:"host" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
	Password string `toml:"password" json:"-"`
	DbIndex  int    `toml:"dbindex" default:"0" json:"dbindex"`
}

type LocalBufferConfiguration struct {
	Path       string                 `toml:"path" json:"path"`
	Encryption []*keyloader.KeyConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
}

type NFSBufferConfiguration struct {
	Host            string                 `toml:"host" json:"host"`
	TargetPartition string                 `toml:"targetPartition" json:"targetPartition"`
	UserID          uint32                 `toml:"userID" json:"userID"`
	GroupID         uint32                 `toml:"groupID" json:"groupID"`
	Encryption      []*keyloader.KeyConfig `toml:"encryption" json:"-" mapstructure:"encryption"`
}

type RunningStorageUnits struct {
	m        *gorpmapper.Mapper
	db       *gorp.DbMap
	cache    cache.Store
	config   Configuration
	Buffers  []BufferUnit
	Storages []StorageUnit
}

func (x RunningStorageUnits) HashLocator(loc string) string {
	salt := []byte(x.config.HashLocatorSalt)
	hashLocator := hex.EncodeToString(pbkdf2.Key([]byte(loc), salt, 4096, 32, sha1.New))
	return hashLocator
}

func (x RunningStorageUnits) FileBuffer() FileBufferUnit {
	for _, u := range x.Buffers {
		if u.BufferType() == CDNBufferTypeFile {
			return u.(FileBufferUnit)
		}
	}
	return nil
}

func (x RunningStorageUnits) LogsBuffer() LogBufferUnit {
	for _, u := range x.Buffers {
		if u.BufferType() == CDNBufferTypeLog {
			return u.(LogBufferUnit)
		}
	}
	return nil
}

func (x RunningStorageUnits) GetBuffer(bufferType sdk.CDNItemType) BufferUnit {
	switch bufferType {
	case sdk.CDNTypeItemStepLog, sdk.CDNTypeItemServiceLog, sdk.CDNTypeItemJobStepLog:
		return x.LogsBuffer()
	default:
		return x.FileBuffer()
	}
}

func (x *RunningStorageUnits) CanSync(unitID string) bool {
	for _, unit := range x.Storages {
		if unit.ID() == unitID {
			return unit.CanSync()
		}
	}
	return false
}

func (x *RunningStorageUnits) FilterNotSyncBackend(ius []sdk.CDNItemUnit) []sdk.CDNItemUnit {
	itemsUnits := make([]sdk.CDNItemUnit, 0, len(ius))
	for _, u := range ius {
		if !x.CanSync(u.UnitID) {
			continue
		}
		itemsUnits = append(itemsUnits, u)
	}
	return itemsUnits
}

func (x *RunningStorageUnits) FilterItemUnitFromBuffer(ius []sdk.CDNItemUnit) []sdk.CDNItemUnit {
	itemsUnits := make([]sdk.CDNItemUnit, 0, len(ius))
	for _, u := range ius {
		if x.IsBuffer(u.UnitID) {
			continue
		}
		itemsUnits = append(itemsUnits, u)
	}
	return itemsUnits
}

func (x *RunningStorageUnits) FilterItemUnitReaderByType(ius []sdk.CDNItemUnit) []sdk.CDNItemUnit {
	// Remove cds backend from getting something that is not a log
	if ius[0].Type != sdk.CDNTypeItemStepLog && ius[0].Type != sdk.CDNTypeItemServiceLog && ius[0].Type != sdk.CDNTypeItemJobStepLog {
		var cdsBackendID string
		for _, unit := range x.Storages {
			if unit.GetDriverName() == "cds" {
				cdsBackendID = unit.ID()
				break
			}
		}

		for i, s := range ius {
			if s.UnitID == cdsBackendID {
				ius = append(ius[:i], ius[i+1:]...)
				break
			}
		}
	}
	return ius
}

func (x *RunningStorageUnits) IsBuffer(id string) bool {
	for _, buf := range x.Buffers {
		if buf.ID() == id {
			return true
		}
	}
	return false
}

type LogConfig struct {
	// Step logs
	StepMaxSize        int64 `toml:"stepMaxSize" default:"15728640" comment:"Max step logs size in bytes (default: 15MB)" json:"stepMaxSize"`
	StepLinesRateLimit int64 `toml:"stepLinesRateLimit" default:"1800" comment:"Number of lines that a worker can send by seconds" json:"stepLinesRateLimit"`
}
