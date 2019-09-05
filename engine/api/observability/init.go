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
	if !cfg.Enable {
		return nil
	}
	traceEnable = true
	var err error
	if traceExporter == nil {
		log.Info("observability> initializing jaeger exporter")
		traceExporter, err = jaeger.NewExporter(jaeger.Options{
			Endpoint:    cfg.Exporter.Jaeger.HTTPCollectorEndpoint, //"http://localhost:14268"
			ServiceName: serviceName,                               //"cds-tracing"
		})
	}
	if err != nil {
		return err
	}
	trace.RegisterExporter(traceExporter)
	trace.ApplyConfig(
		trace.Config{
			DefaultSampler: trace.ProbabilitySampler(cfg.SamplingProbability),
		},
	)

	statsExporter, err = prometheus.NewExporter(prometheus.Options{})
	if err != nil {
		return err
	}
	view.RegisterExporter(statsExporter)
	if cfg.Exporter.Prometheus.ReporteringPeriod == 0 {
		cfg.Exporter.Prometheus.ReporteringPeriod = 10
	}
	view.SetReportingPeriod(time.Duration(cfg.Exporter.Prometheus.ReporteringPeriod) * time.Second)

	return nil
}
