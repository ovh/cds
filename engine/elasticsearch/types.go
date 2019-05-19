package elasticsearch

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
)

const indexNotFoundException = "index_not_found_exception"

// Service is the repostories service
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
}

// Configuration is the vcs configuration structure
type Configuration struct {
	Name string `toml:"name" comment:"Name of this CDS elasticsearch Service\n Enter a name to enable this service" json:"name"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8088" json:"port"`
	} `toml:"http" comment:"######################\n CDS Elasticsearch HTTP Configuration \n######################" json:"http"`
	URL           string `default:"http://localhost:8088" json:"url"`
	ElasticSearch struct {
		URL          string `toml:"url" json:"url"`
		Username     string `toml:"username" json:"username"`
		Password     string `toml:"password" json:"-"`
		IndexEvents  string `toml:"indexEvents" commented:"true" comment:"index to store CDS events" json:"indexEvents"`
		IndexMetrics string `toml:"indexMetrics" commented:"true" comment:"index to store CDS metrics" json:"indexMetrics"`
	} `toml:"elasticsearch" comment:"######################\n CDS ElasticSearch Settings \nSupport for elasticsearch 5.6\n######################" json:"elasticsearch"`
	API service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS Indexes Settings \n######################" json:"api"`
}
