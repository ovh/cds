package metricsservice

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

type MetricProvider interface {
	GetEvents(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	PostEvents(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	GetMetrics(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	PostMetrics(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	GetStatus(name string) sdk.MonitoringStatusLine
}

// Service is the stuct representing a vcs µService
type Service struct {
	service.Common
	Cfg             Configuration
	Router          *api.Router
	metricProviders map[string]MetricProvider
}

// Configuration is the Metrics configuration structure
type Configuration struct {
	Name string `toml:"name" comment:"Name of this CDS Metrics Service\n Enter a name to enable this service" json:"name"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8089" json:"port"`
	} `toml:"http" comment:"######################\n CDS Metrics HTTP Configuration \n######################" json:"http"`
	URL       string                           `default:"http://localhost:8089" json:"url"`
	API       service.APIServiceConfiguration  `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Providers map[string]ProviderConfiguration `toml:"providers" comment:"######################\n CDS Metrics Provider Settings \n######################" json:"providers"`
}

// EndpointConfiguration is the configuration for a Metrics server
type ProviderConfiguration struct {
	Endpoint      string                      `toml:"endpoint" json:"endpoint" comment:"Endpoint of this Provider"`
	ElasticSearch *ElasticSearchConfiguration `toml:"elasticsearch" json:"elasticsearch,omitempty" json:"elasticsearch"`
}

// Configuration is the vcs configuration structure
type ElasticSearchConfiguration struct {
	Username     string `toml:"username" json:"username"`
	Password     string `toml:"password" json:"-"`
	IndexEvents  string `toml:"indexEvents" default:"cds-events" comment:"index to store CDS events" json:"indexEvents"`
	IndexMetrics string `toml:"indexMetrics" default:"cds-metrics" comment:"index to store CDS metrics" json:"indexMetrics"`
}
