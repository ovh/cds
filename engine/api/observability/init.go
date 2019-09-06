package observability

import (
	"time"

	"contrib.go.opencensus.io/exporter/jaeger"
	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/sdk/log"
)

var (
	traceEnable   bool
	traceExporter trace.Exporter
	statsExporter *prometheus.Exporter
)

// Init the opencensus exporter
func Init(cfg Configuration, serviceName string) error {
	var err error

	if cfg.TracingEnabled {
		traceEnable = true
		var err error
		if traceExporter == nil {
			log.Info("observability> initializing jaeger exporter")
			traceExporter, err = jaeger.NewExporter(jaeger.Options{
				Endpoint:    cfg.Exporters.Jaeger.HTTPCollectorEndpoint, //"http://localhost:14268"
				ServiceName: serviceName,                                //"cds-tracing"
			})
		}
		if err != nil {
			return err
		}
		trace.RegisterExporter(traceExporter)
		trace.ApplyConfig(
			trace.Config{
				DefaultSampler: trace.ProbabilitySampler(cfg.Exporters.Jaeger.SamplingProbability),
			},
		)
	}

	if cfg.MetricsEnabled {
		statsExporter, err = prometheus.NewExporter(prometheus.Options{})
		if err != nil {
			return err
		}
		view.RegisterExporter(statsExporter)
		if cfg.Exporters.Prometheus.ReporteringPeriod == 0 {
			cfg.Exporters.Prometheus.ReporteringPeriod = 10
		}
		view.SetReportingPeriod(time.Duration(cfg.Exporters.Prometheus.ReporteringPeriod) * time.Second)
	}

	return nil
}
