package cdn

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// Service is the stuct representing a hooks µService
type Service struct {
	service.Common
	Cfg                 Configuration
	DBConnectionFactory *database.DBConnectionFactory
	Router              *api.Router
	Db                  *gorp.DbMap
	Cache               cache.Store
	Mapper              *gorpmapper.Mapper
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name string        `toml:"name" default:"cds-cdn" comment:"Name of this CDS CDN Service\n Enter a name to enable this service" json:"name"`
	TCP  sdk.TCPServer `toml:"tcp" comment:"######################\n CDS CDN TCP Configuration \n######################" json:"tcp"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8089" json:"port"`
	} `toml:"http" comment:"######################\n CDS CDN HTTP Configuration \n######################" json:"http"`
	URL        string                                 `default:"http://localhost:8089" json:"url" comment:"Private URL for communication with API"`
	PublicTCP  string                                 `toml:"publicTCP" default:"localhost:8090" comment:"Public address to access to CDN TCP server" json:"public_tcp"`
	PublicHTTP string                                 `toml:"publicHTTP" default:"localhost:8089" comment:"Public address to access to CDN HTTP server" json:"public_http"`
	Database   database.DBConfigurationWithEncryption `toml:"database" comment:"################################\n Postgresql Database settings \n###############################" json:"database"`
	Cache      struct {
		TTL   int `toml:"ttl" default:"60" json:"ttl"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
			Password string `toml:"password" json:"-"`
		} `toml:"redis" json:"redis"`
	} `toml:"cache" comment:"######################\n CDS VCS Cache Settings \n######################" json:"cache"`
	API service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Log struct {
		StepMaxSize    int64 `toml:"stepMaxSize" default:"15728640" comment:"Max step logs size in bytes (default: 15MB)" json:"stepMaxSize"`
		ServiceMaxSize int64 `toml:"serviceMaxSize" default:"15728640" comment:"Max service logs size in bytes (default: 15MB)" json:"serviceMaxSize"`
	} `toml:"log" json:"log" comment:"###########################\n Log settings.\n##########################"`
}
