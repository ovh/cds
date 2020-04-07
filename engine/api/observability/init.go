package observability

import (
	"context"
	"time"

	"contrib.go.opencensus.io/exporter/jaeger"
	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	traceExporter     trace.Exporter
	statsExporter     *prometheus.Exporter
	statsHTTPExporter *HTTPExporter
)

type service interface {
	Name() string
	Type() string
}

func serviceName(s service) string {
	return s.Type() + "/" + s.Name()
}

func StatsExporter() *prometheus.Exporter {
	return statsExporter
}

func StatsHTTPExporter() *HTTPExporter {
	return statsHTTPExporter
}

// Init the opencensus exporter
func Init(ctx context.Context, cfg Configuration, s service) (context.Context, error) {
	ctx = ContextWithTag(ctx,
		TagServiceType, s.Type(),
		TagServiceName, s.Name(),
	)

	if cfg.TracingEnabled {
		trace.ApplyConfig(
			trace.Config{
				DefaultSampler: trace.ProbabilitySampler(cfg.Exporters.Jaeger.SamplingProbability),
			},
		)
		log.Info(ctx, "observability> initializing jaeger exporter for %s/%s", s.Type(), s.Name())
		e, err := jaeger.NewExporter(jaeger.Options{
			Endpoint:    cfg.Exporters.Jaeger.HTTPCollectorEndpoint, //"http://localhost:14268"
			ServiceName: serviceName(s),
		})
		if err != nil {
			return ctx, sdk.WithStack(err)
		}
		trace.RegisterExporter(e)
		traceExporter = e
	}

	if cfg.MetricsEnabled {
		if cfg.Exporters.Prometheus.ReporteringPeriod == 0 {
			cfg.Exporters.Prometheus.ReporteringPeriod = 10
		}
		view.SetReportingPeriod(time.Duration(cfg.Exporters.Prometheus.ReporteringPeriod) * time.Second)

		log.Info(ctx, "observability> initializing prometheus exporter for %s/%s", s.Type(), s.Name())

		e, err := prometheus.NewExporter(prometheus.Options{})
		if err != nil {
			return ctx, sdk.WithStack(err)
		}
		view.RegisterExporter(e)
		statsExporter = e

		he := new(HTTPExporter)
		view.RegisterExporter(he)
		statsHTTPExporter = he
	}

	return ctx, nil
}
