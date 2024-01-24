package vcs

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
)

// Service is the stuct representing a vcs µService
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
	Cache  cache.Store
	UI     struct {
		HTTP struct {
			URL string
		}
	}
}

// Configuration is the vcs configuration structure
type Configuration struct {
	Name  string                          `toml:"name" comment:"Name of this CDS VCS Service\n Enter a name to enable this service" json:"name"`
	HTTP  service.HTTPRouterConfiguration `toml:"http" comment:"######################\n CDS VCS HTTP Configuration \n######################" json:"http"`
	URL   string                          `default:"http://localhost:8084" json:"url"`
	API   service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Cache struct {
		TTL   int `toml:"ttl" default:"60" json:"ttl"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
			Password string `toml:"password" json:"-"`
			DbIndex  int    `toml:"dbindex" default:"0" json:"dbindex"`
		} `toml:"redis" json:"redis"`
	} `toml:"cache" comment:"######################\n CDS VCS Cache Settings \n######################" json:"cache"`
	ProxyWebhook string `toml:"proxyWebhook" default:"" commented:"true" comment:"If you want to have a reverse proxy url for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK" json:"proxy_webhook"`
}
