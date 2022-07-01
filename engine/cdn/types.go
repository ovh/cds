package cdn

import (
	"context"
	"sync"
	"time"

	"go.opencensus.io/stats"
	"golang.org/x/time/rate"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/lru"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/websocket"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/log/hook"
)

type handledMessage struct {
	Signature    cdn.Signature
	Msg          hook.Message
	IsTerminated bool
}

// Service is the stuct representing a hooks µService
type Service struct {
	service.Common
	Cfg                 Configuration
	DBConnectionFactory *database.DBConnectionFactory
	Router              *api.Router
	Cache               cache.Store
	LogCache            *lru.Redis
	Mapper              *gorpmapper.Mapper
	Units               *storage.RunningStorageUnits
	WSServer            *websocketServer
	WSBroker            *websocket.Broker
	WSEventsMutex       sync.Mutex
	WSEvents            map[string]sdk.CDNWSEvent
	Metrics             struct {
		tcpServerErrorsCount     *stats.Int64Measure
		tcpServerHitsCount       *stats.Int64Measure
		tcpServerStepLogCount    *stats.Int64Measure
		tcpServerServiceLogCount *stats.Int64Measure
		itemCompletedByGCCount   *stats.Int64Measure
		itemInDatabaseCount      *stats.Int64Measure
		itemPerStorageUnitCount  *stats.Int64Measure
		ItemSize                 *stats.Int64Measure
		ItemToSyncCount          *stats.Int64Measure
		WSClients                *stats.Int64Measure
		WSEvents                 *stats.Int64Measure
		ItemToDelete             *stats.Int64Measure
		ItemUnitToDelete         *stats.Int64Measure
	}
	storageUnitLags          sync.Map
	storageUnitPreviousLags  sync.Map
	storageUnitSizes         sync.Map
	storageUnitPreviousSizes sync.Map
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name               string                          `toml:"name" default:"cds-cdn" comment:"Name of this CDS CDN Service\n Enter a name to enable this service" json:"name"`
	TCP                sdk.TCPServer                   `toml:"tcp" comment:"######################\n CDS CDN TCP Configuration \n######################" json:"tcp"`
	HTTP               service.HTTPRouterConfiguration `toml:"http" comment:"######################\n CDS CDN HTTP Configuration \n######################" json:"http"`
	URL                string                          `default:"http://localhost:8089" json:"url" comment:"Private URL for communication with API"`
	PublicTCP          string                          `toml:"publicTCP" default:"localhost:8090" comment:"Public address to access to CDN TCP server" json:"public_tcp"`
	PublicTCPEnableTLS bool                            `toml:"publicTCPEnableTLS" comment:"Enable TLS on public address to access to CDN TCP server" json:"public_tcp_enable_tls"`
	PublicHTTP         string                          `toml:"publicHTTP" default:"http://localhost:8089" comment:"Public address to access to CDN HTTP server" json:"public_http"`
	Database           database.DBConfiguration        `toml:"database" comment:"################################\n Postgresql Database settings \n###############################" json:"database"`
	Cache              struct {
		TTL     int   `toml:"ttl" default:"60" json:"ttl"`
		LruSize int64 `toml:"lruSize" default:"134217728" json:"lruSize" comment:"Redis LRU cache for logs items in bytes (default: 128MB)"`
		Redis   struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
			Password string `toml:"password" json:"-"`
		} `toml:"redis" json:"redis"`
	} `toml:"cache" comment:"######################\n CDN Cache Settings \n######################" json:"cache"`
	API     service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Log     storage.LogConfig               `toml:"log" json:"log" comment:"###########################\n Log settings.\n##########################"`
	Units   storage.Configuration           `toml:"storageUnits" json:"storageUnits" mapstructure:"storageUnits" comment:"###########################\n Storage Units settings.\n##########################"`
	Metrics struct {
		Frequency int64 `toml:"frequency" default:"30" json:"frequency" comment:"each 30s, metrics are computed"`
	} `toml:"metrics" comment:"######################\n CDN Metrics Settings \n######################" json:"metrics"`
}

type rateLimiter struct {
	limiter *rate.Limiter
	mutex   *sync.Mutex
	ctx     context.Context
}

func NewRateLimiter(ctx context.Context, nbPerSecond float64, burst int) *rateLimiter {
	limit := rate.NewLimiter(rate.Limit(nbPerSecond), burst)
	limit.AllowN(time.Now(), burst)
	return &rateLimiter{ctx: ctx, limiter: limit, mutex: &sync.Mutex{}}
}

func (r *rateLimiter) WaitN(n int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return sdk.WithStack(r.limiter.WaitN(r.ctx, n))
}

type SizeWriter struct {
	Size int64
}

func (s *SizeWriter) Write(data []byte) (n int, err error) {
	s.Size += int64(len(data))
	return len(data), nil
}
