package telemetry

import (
	"sync"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace/propagation"
)

// Attributes recorded on the span for the requests.
// Only trace exporters will need them.
const (
	HostAttribute      = "http.host"
	MethodAttribute    = "http.method"
	PathAttribute      = "http.path"
	UserAgentAttribute = "http.user_agent"
)

// DefaultFormat used by observability as: observability.DefaultFormat.SpanContextToRequest
var DefaultFormat propagation.HTTPFormat = &b3.HTTPFormat{}

// Configuration is the global tracing configuration
type Configuration struct {
	TracingEnabled bool `toml:"tracingEnabled" json:"tracingEnabled"`
	Exporters      struct {
		Jaeger struct {
			ServiceName         string  `toml:"ServiceName" default:"" json:"serviceName"`
			CollectorEndpoint   string  `toml:"collectorEndpoint" default:"http://localhost:14268/api/traces" json:"collectorEndpoint"`
			SamplingProbability float64 `toml:"samplingProbability" json:"metricSamplingProbability"`
		} `json:"jaeger"`
		Prometheus struct {
			ReporteringPeriod int `toml:"ReporteringPeriod" default:"10" json:"reporteringPeriod"`
		} `json:"prometheus"`
	} `json:"exporter"`
}

var (
	// DefaultSizeDistribution 25k, 100k, 250, 500, 1M, 1.5M, 5M, 10M,
	DefaultSizeDistribution = view.Distribution(25*1024, 100*1024, 250*1024, 500*1024, 1024*1024, 1.5*1024*1024, 5*1024*1024, 10*1024*1024)
	// DefaultLatencyDistribution 100ms, ...
	DefaultLatencyDistribution = view.Distribution(100, 200, 300, 400, 500, 750, 1000, 2000, 5000)
)

const (
	Host       = "http.host"
	StatusCode = "http.status"
	Path       = "http.path"
	Method     = "http.method"
	Handler    = "http.handler"
	RequestID  = "http.request-id"
)

type ExposedView struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Dimension   string   `json:"dimension"`
	Aggregation string   `json:"aggregagtion"`
}

type HTTPExporter struct {
	*prometheus.Exporter `json:"-"`
	ExposedViews         []ExposedView `json:"-"`
	exposedViewMutex     sync.Mutex    `json:"-"`
	Views                []HTTPExporterView
}

type HTTPExporterView struct {
	Name  string            `json:"name"`
	Tags  map[string]string `json:"tags"`
	Value float64           `json:"value"`
	Date  time.Time         `json:"date"`
}

type contextKey int

const (
	contextTraceExporter contextKey = iota
	contextStatsExporter
)

type Service interface {
	Name() string
	Type() string
}

// B3 headers that OpenCensus understands.
const (
	TraceIDHeader = "X-B3-TraceId"
	SpanIDHeader  = "X-B3-SpanId"
	SampledHeader = "X-B3-Sampled"

	ContextTraceIDHeader contextKey = iota
	ContextSpanIDHeader
	ContextSampledHeader
	ContextMainSpan
)

//Options is the options struct for a new tracing span
type Options struct {
	Init     bool
	Name     string
	SpanKind int
}
