package elasticsearch

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
)

// Service is the repostories service
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
}

// Configuration is the vcs configuration structure
type Configuration struct {
	Name string `toml:"name" comment:"Name of this CDS elasticsearch Service"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1"`
		Port int    `toml:"port" default:"8088"`
	} `toml:"http" comment:"######################\n CDS Elasticsearch HTTP Configuration \n######################"`
	URL           string `default:"http://localhost:8088"`
	ElasticSearch struct {
		URL      string `toml:"url"`
		Username string `toml:"username"`
		Password string `toml:"password"`
		Index    string `toml:"index"`
	} `toml:"elasticsearch" comment:"######################\n CDS ElasticSearch Settings \n######################"`
	API service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################"`
}
