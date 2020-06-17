package cdn

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// Service is the stuct representing a hooks µService
type Service struct {
	service.Common
	Cfg Configuration
	//Router *api.Router
	Db           *gorp.DbMap
	Cache        cache.Store
	ChanMessages chan handledMessage
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name string        `toml:"name" default:"cds-cdn" comment:"Name of this CDS CDN Service\n Enter a name to enable this service" json:"name"`
	TCP  sdk.TCPServer `toml:"tcp" comment:"######################\n CDS CDN TCP Configuration \n######################" json:"tcp"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8089" json:"port"`
	} `toml:"http" comment:"######################\n CDS CDN HTTP Configuration \n######################" json:"http"`
	URL        string                          `default:"http://localhost:8089" json:"url" comment:"Private URL for communication with API"`
	PublicTCP  string                          `toml:"publicTCP" comment:"Public address to access to CDN TCP server"`
	PublicHTTP string                          `toml:"publicHTTP" comment:"Public address to access to CDN HTTP server"`
	API        service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Log        struct {
		StepMaxSize    int64 `toml:"stepMaxSize" default:"15728640" comment:"Max step logs size in bytes (default: 15MB)" json:"stepMaxSize"`
		ServiceMaxSize int64 `toml:"serviceMaxSize" default:"15728640" comment:"Max service logs size in bytes (default: 15MB)" json:"serviceMaxSize"`
	} `toml:"log" json:"log" comment:"###########################\n Log settings.\n##########################"`
}
