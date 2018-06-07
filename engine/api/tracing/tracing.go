package tracing

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/feature"
)

var traceEnable bool

// Init the tracer
func Init(cfg Configuration) error {
	if !cfg.Enable {
		return nil
	}
	traceEnable = true
	exporter, err := jaeger.NewExporter(jaeger.Options{
		Endpoint:    cfg.Exporter.Jaeger.HTTPCollectorEndpoint, //"http://localhost:14268"
		ServiceName: cfg.Exporter.Jaeger.ServiceName,           //"cds-tracing"
	})
	if err != nil {
		return err
	}
	trace.RegisterExporter(exporter)
	trace.ApplyConfig(
		trace.Config{
			DefaultSampler: trace.ProbabilitySampler(cfg.SamplingProbability),
		},
	)

	return nil
}

// Start may start a tracing span
func Start(ctx context.Context, w http.ResponseWriter, req *http.Request, opt Options, db gorp.SqlExecutor, store cache.Store) (context.Context, error) {
	if !traceEnable || !opt.Enable {
		return ctx, nil
	}

	tags := []trace.Attribute{}
	if opt.Worker != nil {
		tags = append(tags, trace.StringAttribute("worker", opt.Worker.Name))
	}
	if opt.Hatchery != nil {
		tags = append(tags, trace.StringAttribute("hatchery", opt.Hatchery.Name))
	}
	if opt.User != nil {
		tags = append(tags, trace.StringAttribute("user", opt.User.Username))
	}

	var span *trace.Span
	rootSpanContext, hasSpanContext := defaultFormat.SpanContextFromRequest(req)

	type setupFuncSpan func(s *trace.Span, r *http.Request, sc *trace.SpanContext)
	var setupSpan = []setupFuncSpan{
		func(s *trace.Span, r *http.Request, sc *trace.SpanContext) {
			s.AddAttributes(
				trace.StringAttribute(PathAttribute, r.URL.Path),
				trace.StringAttribute(HostAttribute, r.URL.Host),
				trace.StringAttribute(MethodAttribute, r.Method),
				trace.StringAttribute(UserAgentAttribute, r.UserAgent()),
			)
		},
	}
	if hasSpanContext {
		setupSpan = append(setupSpan, func(s *trace.Span, r *http.Request, sc *trace.SpanContext) {
			s.AddLink(trace.Link{
				TraceID:    rootSpanContext.TraceID,
				SpanID:     rootSpanContext.SpanID,
				Type:       trace.LinkTypeChild,
				Attributes: nil,
			})
			spanContextToReponse(*sc, r, w)
		})
	} else {
		setupSpan = append(setupSpan, func(s *trace.Span, r *http.Request, sc *trace.SpanContext) {
			defaultFormat.SpanContextToRequest(*sc, r)
			spanContextToReponse(*sc, r, w)
		})
	}

	pkey, ok := findPrimaryKeyFromRequest(req, db, store)
	if pkey != "" {
		tags = append(tags, trace.StringAttribute("project_key", pkey))
	}

	switch {
	case ok && feature.IsEnabled(store, feature.FeatEnableTracing, pkey):
		ctx, span = trace.StartSpan(ctx, opt.Name,
			trace.WithSampler(trace.AlwaysSample()),
			trace.WithSpanKind(trace.SpanKindServer))
	default:
		ctx, span = trace.StartSpan(ctx, opt.Name,
			trace.WithSpanKind(trace.SpanKindServer))
	}

	var sc trace.SpanContext
	if !hasSpanContext {
		sc = span.SpanContext()
	}
	span.AddAttributes(tags...)
	for _, f := range setupSpan {
		f(span, req, &sc)
	}

	return ctx, nil
}

// End may close a tracing span
func End(ctx context.Context, w http.ResponseWriter, req *http.Request) (context.Context, error) {
	span := trace.FromContext(ctx)
	if span == nil {
		return ctx, nil
	}

	span.End()
	return ctx, nil
}
