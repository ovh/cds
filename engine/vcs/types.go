package vcs

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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
		TTL   int           `toml:"ttl" default:"60" json:"ttl"`
		Redis sdk.RedisConf `toml:"redis" json:"redis"`
	} `toml:"cache" comment:"######################\n CDS VCS Cache Settings \n######################" json:"cache"`
	Forge struct {
		ConnectionPoolSize  int64 `toml:"connectionPoolSize" comment:"Number of connections this service keeps open to a forge. Concurrent requests are spread over them. Raise it when the forge accepts more concurrency, lower it when it rate limits simultaneous requests." json:"connectionPoolSize" commented:"true" default:"8"`
		MaxIdleConnsPerHost int64 `toml:"maxIdleConnsPerHost" comment:"Number of idle connections kept per forge host, for each connection above. Below this, a request opens a new connection instead of reusing one." json:"maxIdleConnsPerHost" commented:"true" default:"32"`
	} `toml:"forge" comment:"######################\n CDS VCS forge connection settings \n######################" json:"forge"`
	ProxyWebhook string `toml:"proxyWebhook" default:"" commented:"true" comment:"If you want to have a reverse proxy url for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK" json:"proxy_webhook"`
}
