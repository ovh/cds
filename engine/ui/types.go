package ui

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
)

// Service is the stuct representing a ui µService
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name      string `toml:"name" comment:"Name of this CDS UI Service\n Enter a name to enable this service" json:"name"`
	Staticdir string `toml:"staticdir" comment:"This directory must contains index.html file and other ui files (css, js...) from ui.tar.gz artifact." json:"staticdir"`
	HTTP      struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8080" json:"port"`
	} `toml:"http" comment:"######################\n CDS UI HTTP Configuration \n######################" json:"http"`
	URL      string                          `default:"http://localhost:8080" json:"url"`
	API      service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	HooksURL string                          `toml:"hooksURL" comment:"Hooks µService URL" default:"http://localhost:8083" json:"hooksURL"`
}
