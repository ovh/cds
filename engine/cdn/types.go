package cdn

import (
	"encoding/json"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// Service is the stuct representing a hooks µService
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
	Db     *gorp.DbMap
	Cache  cache.Store
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name string        `toml:"name" default:"cds-cdn" comment:"Name of this CDS CDN Service\n Enter a name to enable this service" json:"name"`
	TCP  sdk.TCPServer `toml:"tcp" comment:"######################\n CDS CDN TCP Configuration \n######################" json:"tcp"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8089" json:"port"`
	} `toml:"http" comment:"######################\n CDS CDN HTTP Configuration \n######################" json:"http"`
	URL string                          `default:"http://localhost:8089" json:"url"`
	API service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Log struct {
		StepMaxSize    int64 `toml:"stepMaxSize" default:"15728640" comment:"Max step logs size in bytes (default: 15MB)" json:"stepMaxSize"`
		ServiceMaxSize int64 `toml:"serviceMaxSize" default:"15728640" comment:"Max service logs size in bytes (default: 15MB)" json:"serviceMaxSize"`
	} `toml:"log" json:"log" comment:"###########################\n Log settings.\n##########################"`
}

func (c *Configuration) Unmarshal(configService sdk.ServiceConfig) error {
	b, err := json.Marshal(configService)
	if err != nil {
		return sdk.WithStack(err)
	}
	if err := json.Unmarshal(b, c); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
