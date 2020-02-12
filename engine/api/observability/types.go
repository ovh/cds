package observability

// Attributes recorded on the span for the requests.
// Only trace exporters will need them.
const (
	HostAttribute       = "http.host"
	MethodAttribute     = "http.method"
	PathAttribute       = "http.path"
	UserAgentAttribute  = "http.user_agent"
	StatusCodeAttribute = "http.status_code"
)

// Configuration is the global tracing configuration
type Configuration struct {
	MetricsEnabled bool `toml:"metricsEnabled" json:"metricsEnabled"`
	TracingEnabled bool `toml:"tracingEnabled" json:"tracingEnabled"`
	Exporters      struct {
		Jaeger struct {
			HTTPCollectorEndpoint string  `toml:"HTTPCollectorEndpoint" default:"http://localhost:14268" json:"httpCollectorEndpoint"`
			SamplingProbability   float64 `toml:"samplingProbability" json:"metricSamplingProbability"`
		} `json:"jaeger"`
		Prometheus struct {
			ReporteringPeriod int `toml:"ReporteringPeriod" default:"10" json:"reporteringPeriod"`
		} `json:"prometheus"`
	} `json:"exporter"`
}

//Options is the options struct for a new tracing span
type Options struct {
	Init     bool
	Name     string
	Enable   bool
	SpanKind int
}
