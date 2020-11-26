package ui

import (
	"net/http"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
)

// Service is the stuct representing a ui µService
type Service struct {
	service.Common
	Cfg     Configuration
	Router  *api.Router
	Server  *http.Server
	HTMLDir string
	DocsDir string
}

// Configuration is the ui configuration structure
type Configuration struct {
	Name      string `toml:"name" comment:"Name of this CDS UI Service\n Enter a name to enable this service" json:"name"`
	Staticdir string `toml:"staticdir" default:"./ui_static_files" comment:"This directory must contain the dist directory." json:"staticdir"`
	BaseURL   string `toml:"baseURL" commented:"true" comment:"If you expose CDS UI with https://your-domain.com/ui, enter the value '/ui/'. Optional" json:"baseURL"`
	DeployURL string `toml:"deployURL" commented:"true" comment:"You can start CDS UI proxy on a sub path like https://your-domain.com/ui with value '/ui' (the value should not be given when the sub path is added by a proxy in front of CDS). Optional" json:"deployURL"`
	SentryURL string `toml:"sentryURL" commented:"true" comment:"Sentry URL. Optional" json:"-"`
	HTTP      struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8080" json:"port"`
	} `toml:"http" comment:"######################\n CDS UI HTTP Configuration \n######################" json:"http"`
	URL      string                          `toml:"url" comment:"Public URL of this UI service." default:"http://localhost:8080" json:"url"`
	API      service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	HooksURL string                          `toml:"hooksURL" comment:"Hooks µService URL" default:"http://localhost:8083" json:"hooksURL"`
	CDNURL   string                          `toml:"cdnURL" comment:"CDN µService URL" default:"http://localhost:8089" json:"cdnURL"`
}
