package cdn

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

type handledMessage struct {
	Signature log.Signature
	Msg       hook.Message
	Line      int64
	Status    string
}

// Service is the stuct representing a hooks µService
type Service struct {
	service.Common
	Cfg                 Configuration
	DBConnectionFactory *database.DBConnectionFactory
	Router              *api.Router
	Cache               cache.Store
	Mapper              *gorpmapper.Mapper
	Units               *storage.RunningStorageUnits
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name string        `toml:"name" default:"cds-cdn" comment:"Name of this CDS CDN Service\n Enter a name to enable this service" json:"name"`
	TCP  sdk.TCPServer `toml:"tcp" comment:"######################\n CDS CDN TCP Configuration \n######################" json:"tcp"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8089" json:"port"`
	} `toml:"http" comment:"######################\n CDS CDN HTTP Configuration \n######################" json:"http"`
	URL                 string                                 `default:"http://localhost:8089" json:"url" comment:"Private URL for communication with API"`
	PublicTCP           string                                 `toml:"publicTCP" default:"localhost:8090" comment:"Public address to access to CDN TCP server" json:"public_tcp"`
	PublicHTTP          string                                 `toml:"publicHTTP" default:"localhost:8089" comment:"Public address to access to CDN HTTP server" json:"public_http"`
	EnableLogProcessing bool                                   `toml:"enableLogProcessing" comment:"Enable CDN preview feature that will index logs (this require a database)" json:"enableDatabaseFeatures"`
	Database            database.DBConfigurationWithEncryption `toml:"database" comment:"################################\n Postgresql Database settings \n###############################" json:"database"`
	Cache               struct {
		TTL   int `toml:"ttl" default:"60" json:"ttl"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
			Password string `toml:"password" json:"-"`
		} `toml:"redis" json:"redis"`
	} `toml:"cache" comment:"######################\n CDN Cache Settings \n######################" json:"cache"`
	API service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Log struct {
		StepMaxSize    int64 `toml:"stepMaxSize" default:"15728640" comment:"Max step logs size in bytes (default: 15MB)" json:"stepMaxSize"`
		ServiceMaxSize int64 `toml:"serviceMaxSize" default:"15728640" comment:"Max service logs size in bytes (default: 15MB)" json:"serviceMaxSize"`
	} `toml:"log" json:"log" comment:"###########################\n Log settings.\n##########################"`
	NbJobLogsGoroutines     int64                 `toml:"nbJobLogsGoroutines" default:"5" comment:"Number of workers that dequeue the job log queue" json:"nbJobLogsGoroutines"`
	NbServiceLogsGoroutines int64                 `toml:"nbServiceLogsGoroutines" default:"5" comment:"Number of workers that dequeue the service log queue" json:"nbServiceLogsGoroutines"`
	Units                   storage.Configuration `toml:"storageUnits"  json:"storageUnits" mapstructure:"storageUnits"`
}
