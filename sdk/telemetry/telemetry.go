package telemetry

import (
	"context"
	"fmt"
	"time"

	"contrib.go.opencensus.io/exporter/jaeger"
	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
)

func serviceName(s Service) string {
	return s.Type() + "/" + s.Name()
}

func StatsExporter(ctx context.Context) *HTTPExporter {
	i := ctx.Value(contextStatsExporter)
	exp, ok := i.(*HTTPExporter)
	if ok {
		return exp
	}
	return nil
}

func TraceExporter(ctx context.Context) trace.Exporter {
	i := ctx.Value(contextTraceExporter)
	exp, ok := i.(trace.Exporter)
	if ok {
		return exp
	}
	return nil
}

func ContextWithTelemetry(from, to context.Context) context.Context {
	se := StatsExporter(from)
	te := TraceExporter(from)
	if se != nil {
		to = context.WithValue(to, contextStatsExporter, se)
	}
	if te != nil {
		to = context.WithValue(to, contextTraceExporter, te)
	}
	return to
}

// Init the opencensus exporter
func Init(ctx context.Context, cfg Configuration, s Service) (context.Context, error) {
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
		var svcName = cfg.Exporters.Jaeger.ServiceName
		if svcName == "" {
			svcName = serviceName(s)
		}
		log.Info(ctx, "observability> initializing jaeger exporter for %q on %q", svcName, cfg.Exporters.Jaeger.CollectorEndpoint)
		e, err := jaeger.NewExporter(jaeger.Options{
			CollectorEndpoint: cfg.Exporters.Jaeger.CollectorEndpoint,
			ServiceName:       svcName,
		})
		if err != nil {
			return ctx, errors.WithStack(err)
		}
		trace.RegisterExporter(e)
		ctx = context.WithValue(ctx, contextTraceExporter, e)
	}

	if cfg.Exporters.Prometheus.ReporteringPeriod == 0 {
		cfg.Exporters.Prometheus.ReporteringPeriod = 10
	}
	view.SetReportingPeriod(time.Duration(cfg.Exporters.Prometheus.ReporteringPeriod) * time.Second)

	log.Info(ctx, "observability> initializing prometheus exporter for %q", serviceName(s))

	e, err := prometheus.NewExporter(prometheus.Options{})
	if err != nil {
		return ctx, errors.WithStack(err)
	}
	view.RegisterExporter(e)
	he := new(HTTPExporter)
	he.Exporter = e
	view.RegisterExporter(he)
	ctx = context.WithValue(ctx, contextStatsExporter, he)

	return ctx, nil
}

// Tags contants
const (
	TagGoroutine          = "goroutine"
	TagHostname           = "hostname"
	TagJob                = "job"
	TagRepository         = "repository"
	TagVCSServer          = "vcs_server"
	TagPercentil          = "percentil"
	TagPermission         = "permission"
	TagPipeline           = "pipeline"
	TagPipelineDeep       = "pipeline_deep"
	TagPipelineID         = "pipeline_id"
	TagProjectKey         = "project_key"
	TagServiceName        = "service_name"
	TagServiceType        = "service_type"
	TagStatus             = "status"
	TagStorage            = "storage"
	TagType               = "type"
	TagWorker             = "worker"
	TagWorkflow           = "workflow"
	TagWorkflowNode       = "workflow_node"
	TagWorkflowNodeJobRun = "workflow_node_job_run"
	TagWorkflowNodeRun    = "workflow_node_run"
	TagWorkflowRun        = "workflow_run"
	TagWorkflowRunNumber  = "run_number"
	TagEventID            = "event_id"
)

func ContextWithTag(ctx context.Context, s ...interface{}) context.Context {
	if len(s)%2 != 0 {
		panic("tags key/value are incorrect")
	}
	var tags []tag.Mutator
	for i := 0; i < len(s)-1; i = i + 2 {
		k, err := tag.NewKey(s[i].(string))
		if err != nil {
			log.Error(ctx, "ContextWithTag> %v", err)
			continue
		}
		tags = append(tags, tag.Upsert(k, fmt.Sprintf("%v", s[i+1])))
	}
	ctx, _ = tag.New(ctx, tags...)
	return ctx
}

func ContextGetTags(ctx context.Context, s ...string) []tag.Mutator {
	m := tag.FromContext(ctx)
	var tags []tag.Mutator

	for i := 0; i < len(s); i++ {
		k, err := tag.NewKey(s[i])
		if err != nil {
			log.Error(ctx, "ContextGetTags> %v", err)
			continue
		}
		val, ok := m.Value(k)
		if ok {
			tags = append(tags, tag.Upsert(k, val))
		}
	}
	return tags
}

func MustNewKey(s string) tag.Key {
	k, err := tag.NewKey(s)
	if err != nil {
		panic(err)
	}
	return k
}
