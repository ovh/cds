package observability

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/sdk/tracingutils"
)

// New may start a tracing span
func New(ctx context.Context, s service, name string, sampler trace.Sampler, spanKind int) (context.Context, *trace.Span) {
	if traceExporter == nil {
		return ctx, nil
	}
	return trace.StartSpan(ctx, name,
		trace.WithSampler(sampler),
		trace.WithSpanKind(spanKind))
}

// Start may start a tracing span
func Start(ctx context.Context, s service, w http.ResponseWriter, req *http.Request, opt Options, db gorp.SqlExecutor, store cache.Store) (context.Context, error) {
	if traceExporter == nil {
		return ctx, nil
	}

	tags := []trace.Attribute{}

	var span *trace.Span
	rootSpanContext, hasSpanContext := tracingutils.DefaultFormat.SpanContextFromRequest(req)

	var pkey string
	var ok bool
	if db != nil && store != nil {
		pkey, ok = findPrimaryKeyFromRequest(ctx, req, db, store)
		if pkey != "" {
			tags = append(tags, trace.StringAttribute("project_key", pkey))
		}
	}

	var traceOpts = []trace.StartOption{
		trace.WithSpanKind(trace.SpanKindServer),
	}

	var sampler trace.Sampler
	switch {
	case ok && feature.IsEnabled(ctx, store, feature.FeatEnableTracing, pkey):
		sampler = trace.AlwaysSample()
	case hasSpanContext && rootSpanContext.IsSampled():
		sampler = trace.AlwaysSample()
	}

	if sampler != nil {
		traceOpts = append(traceOpts, trace.WithSampler(sampler))
	}

	if hasSpanContext {
		ctx, span = trace.StartSpanWithRemoteParent(ctx, opt.Name, rootSpanContext, traceOpts...)
		span.AddLink(
			trace.Link{
				TraceID: rootSpanContext.TraceID,
				SpanID:  rootSpanContext.SpanID,
				Type:    trace.LinkTypeChild,
			},
		)
	} else {
		ctx, span = trace.StartSpan(ctx, opt.Name, traceOpts...)
	}

	span.AddAttributes(tags...)
	span.AddAttributes(
		trace.StringAttribute(PathAttribute, req.URL.Path),
		trace.StringAttribute(HostAttribute, req.URL.Host),
		trace.StringAttribute(MethodAttribute, req.Method),
		trace.StringAttribute(UserAgentAttribute, req.UserAgent()),
	)

	ctx = tracingutils.SpanContextToContext(ctx, span.SpanContext())
	ctx = ContextWithTag(ctx,
		TagServiceType, s.Type(),
		TagServiceName, s.Name(),
	)
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
