package cdn

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/service"
)

// Service is the stuct representing a hooks µService
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
	Cache  cache.Store
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name string `toml:"name" comment:"Name of this CDS CDN Service\n Enter a name to enable this service" json:"name"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8087" json:"port"`
	} `toml:"http" comment:"######################\n CDS Hooks HTTP Configuration \n######################" json:"http"`
	URL   string                          `default:"http://localhost:8087" json:"url"`
	API   service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Cache struct {
		TTL   int `toml:"ttl" default:"60" json:"ttl"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax! <clustername>@sentinel1:26379,sentinel2:26379,sentinel3:26379" json:"host"`
			Password string `toml:"password" json:"-"`
		} `toml:"redis" comment:"Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup" json:"redis"`
	} `toml:"cache" comment:"######################\n CDS Hooks Cache Settings \n######################" json:"cache"`
}
