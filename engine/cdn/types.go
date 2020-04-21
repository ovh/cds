package cdn

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
)

// Service is the stuct representing a hooks µService
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name string `toml:"name" default:"cds-cdn" comment:"Name of this CDS CDN Service\n Enter a name to enable this service" json:"name"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8089" json:"port"`
	} `toml:"http" comment:"######################\n CDS CDN HTTP Configuration \n######################" json:"http"`
	URL string                          `default:"http://localhost:8087" json:"url"`
	API service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
}
