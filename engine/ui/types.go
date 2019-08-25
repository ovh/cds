package ui

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
)

// Service is the stuct representing a ui µService
type Service struct {
	service.Common
	Cfg     Configuration
	Router  *api.Router
	HTMLDir string
}

// Configuration is the ui configuration structure
type Configuration struct {
	Name      string `toml:"name" comment:"Name of this CDS UI Service\n Enter a name to enable this service" json:"name"`
	Staticdir string `toml:"staticdir" default:"./ui_static_files" comment:"This directory must contains index.html file and other ui files (css, js...) from ui.tar.gz artifact." json:"staticdir"`
	BaseURL   string `toml:"baseURL" default:"/" comment:"Base URL. If you expose CDS UI with https://your-domain.com/ui, enter the value '/ui'" json:"baseURL"`
	SentryURL string `toml:"sentryURL" default:"" comment:"Sentry URL. Optional" json:"sentryURL"`
	HTTP      struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8080" json:"port"`
	} `toml:"http" comment:"######################\n CDS UI HTTP Configuration \n######################" json:"http"`
<<<<<<< HEAD
	URL      string                          `toml:"url" comment:"URL of this UI service" default:"http://localhost:8080" json:"url"`
=======
	URL      string                          `default:"http://localhost:8080" json:"url"`
>>>>>>> 21d319561ff684fb3763ac99f9d183659dcac8d6
	API      service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	HooksURL string                          `toml:"hooksURL" comment:"Hooks µService URL" default:"http://localhost:8083" json:"hooksURL"`
}
