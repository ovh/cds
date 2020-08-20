package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"go.opencensus.io/trace"
)

// New may start a tracing span
func New(ctx context.Context, s Service, name string, sampler trace.Sampler, spanKind int) (context.Context, *trace.Span) {
	exp := TraceExporter(ctx)
	if exp == nil {
		return ctx, nil
	}
	ctx, span := trace.StartSpan(ctx, name,
		trace.WithSampler(sampler),
		trace.WithSpanKind(spanKind))
	ctx = SpanContextToContext(ctx, span.SpanContext())
	ctx = ContextWithTag(ctx,
		TagServiceType, s.Type(),
		TagServiceName, s.Name(),
	)
	return ctx, span
}

// Start may start a tracing span
func Start(ctx context.Context, s Service, w http.ResponseWriter, req *http.Request, opt Options) (context.Context, error) {
	exp := TraceExporter(ctx)
	if exp == nil {
		return ctx, nil
	}

	tags := []trace.Attribute{}

	var span *trace.Span
	rootSpanContext, hasSpanContext := DefaultFormat.SpanContextFromRequest(req)

	var traceOpts = []trace.StartOption{
		trace.WithSpanKind(trace.SpanKindServer),
	}

	var sampler trace.Sampler
	if hasSpanContext && rootSpanContext.IsSampled() {
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

	ctx = context.WithValue(ctx, ContextMainSpan, span)

	ctx = SpanContextToContext(ctx, span.SpanContext())
	ctx = ContextWithTag(ctx,
		TagServiceType, s.Type(),
		TagServiceName, s.Name(),
	)
	return ctx, nil
}

// End may close a tracing span
func End(ctx context.Context, w http.ResponseWriter, req *http.Request) (context.Context, error) {
	span := MainSpan(ctx)
	if span == nil {
		return ctx, nil
	}

	span.End()
	return ctx, nil
}

// LinkTo a traceID
func LinkTo(ctx context.Context, traceID [16]byte) {
	s := Current(ctx)
	if s == nil {
		return
	}

	s.AddLink(
		trace.Link{
			TraceID: traceID,
		},
	)
}

// Current return the current span
func Current(ctx context.Context, tags ...trace.Attribute) *trace.Span {
	if ctx == nil {
		return nil
	}
	span := trace.FromContext(ctx)
	if span == nil {
		return nil
	}
	if len(tags) > 0 {
		span.AddAttributes(tags...)
	}
	return span
}

// Tag is helper function to instantiate trace.Attribute
func Tag(key string, value interface{}) trace.Attribute {
	return trace.StringAttribute(key, fmt.Sprintf("%v", value))
}

func MainSpan(ctx context.Context) *trace.Span {
	spanI := ctx.Value(ContextMainSpan)
	if spanI == nil {
		return nil
	}

	rootSpan, ok := spanI.(*trace.Span)
	if !ok {
		return nil
	}

	return rootSpan
}

func SpanFromMain(ctx context.Context, name string, tags ...trace.Attribute) (context.Context, func()) {
	rootSpan := MainSpan(ctx)
	if rootSpan == nil {
		return ctx, func() {}
	}
	rootSpanContext := rootSpan.SpanContext()

	var traceOpts = []trace.StartOption{}

	var sampler trace.Sampler
	if rootSpanContext.IsSampled() {
		sampler = trace.AlwaysSample()
	}

	if sampler != nil {
		traceOpts = append(traceOpts, trace.WithSampler(sampler))
	}

	ctx, span := trace.StartSpanWithRemoteParent(ctx, name, rootSpanContext, traceOpts...)
	span.AddLink(trace.Link{
		TraceID: rootSpanContext.TraceID,
		SpanID:  rootSpanContext.SpanID,
	})
	span.AddAttributes(tags...)

	return ctx, span.End
}

// Span start a new span from the parent context
func Span(ctx context.Context, name string, tags ...trace.Attribute) (context.Context, func()) {
	// log.Debug("# %s - begin", name)
	if ctx == nil {
		return context.Background(), func() {}
	}
	var span *trace.Span
	ctx, span = trace.StartSpan(ctx, name)
	if len(tags) > 0 {
		span.AddAttributes(tags...)
	}
	ctx = SpanContextToContext(ctx, span.SpanContext())
	return ctx, func() {
		// log.Debug("# %s - end", name)
		span.End()
	}
}
