package tracing

import (
	"github.com/ovh/cds/sdk"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/trace/propagation"
)

// Attributes recorded on the span for the requests.
// Only trace exporters will need them.
const (
	HostAttribute       = "http.host"
	MethodAttribute     = "http.method"
	PathAttribute       = "http.path"
	UserAgentAttribute  = "http.user_agent"
	StatusCodeAttribute = "http.status_code"
)

var defaultFormat propagation.HTTPFormat = &b3.HTTPFormat{}

// Configuration is the global tracing configuration
type Configuration struct {
	Enable   bool
	Exporter struct {
		Jaeger struct {
			HTTPCollectorEndpoint string `toml:"HTTPCollectorEndpoint" default:"http://localhost:14268"`
			ServiceName           string `toml:"serviceName" default:"cds-api"`
		}
	}
	SamplingProbability float64
}

//Options is the options struct for a new tracing span
type Options struct {
	Init     bool
	Name     string
	Enable   bool
	User     *sdk.User
	Worker   *sdk.Worker
	Hatchery *sdk.Hatchery
}
