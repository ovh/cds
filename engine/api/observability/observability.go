package observability

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

// Start may start a tracing span
func Start(ctx context.Context, s telemetry.Service, w http.ResponseWriter, req *http.Request, opt telemetry.Options, m *gorpmapper.Mapper, db gorp.SqlExecutor, store cache.Store) (context.Context, error) {
	exp := telemetry.TraceExporter(ctx)
	if exp == nil {
		return ctx, nil
	}

	tags := []trace.Attribute{}

	var span *trace.Span
	rootSpanContext, hasSpanContext := telemetry.DefaultFormat.SpanContextFromRequest(req)

	var pkey string
	if db != nil && store != nil {
		pkey, _ = findPrimaryKeyFromRequest(ctx, req, db, store)
		if pkey != "" {
			tags = append(tags, trace.StringAttribute("project_key", pkey))
		}
	}

	var traceOpts = []trace.StartOption{
		trace.WithSpanKind(trace.SpanKindServer),
	}

	mapVars := map[string]string{
		"trace":                      opt.Name,
		"project_key":                pkey,
		telemetry.PathAttribute:      req.URL.Path,
		telemetry.HostAttribute:      req.URL.Host,
		telemetry.MethodAttribute:    req.Method,
		telemetry.UserAgentAttribute: req.UserAgent(),
	}

	var sampler trace.Sampler
	switch {
	case featureflipping.IsEnabled(ctx, m, db, "tracing", mapVars):
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
		trace.StringAttribute(telemetry.PathAttribute, req.URL.Path),
		trace.StringAttribute(telemetry.HostAttribute, req.URL.Host),
		trace.StringAttribute(telemetry.MethodAttribute, req.Method),
		trace.StringAttribute(telemetry.UserAgentAttribute, req.UserAgent()),
	)

	log.Debug("# %s saving main span: %+v", opt.Name, span)
	ctx = context.WithValue(ctx, telemetry.ContextMainSpan, span)

	ctx = telemetry.SpanContextToContext(ctx, span.SpanContext())
	ctx = telemetry.ContextWithTag(ctx,
		telemetry.TagServiceType, s.Type(),
		telemetry.TagServiceName, s.Name(),
	)
	return ctx, nil
}

func findPrimaryKeyFromRequest(ctx context.Context, req *http.Request, db gorp.SqlExecutor, store cache.Store) (string, bool) {
	vars := mux.Vars(req)
	pkey := vars["key"]
	if pkey == "" {
		pkey = vars["permProjectKey"]
	}

	if pkey == "" {
		id, _ := strconv.ParseInt(vars["id"], 10, 64)
		//The ID found may be a node run job, let's try to find the project key behing
		if id <= 0 {
			id, _ = strconv.ParseInt(vars["permJobID"], 10, 64)
		}
		if id != 0 {
			var err error
			cacheKey := cache.Key("api:FindProjetKeyForNodeRunJob:", fmt.Sprintf("%v", id))
			find, errGet := store.Get(cacheKey, &pkey)
			if errGet != nil {
				log.Error(ctx, "cannot get from cache %s: %v", cacheKey, errGet)
			}
			if !find {
				pkey, err = findProjetKeyForNodeRunJob(ctx, db, id)
				if err != nil {
					log.Error(ctx, "tracingMiddleware> %v", err)
					return "", false
				}
				if err := store.SetWithTTL(cacheKey, pkey, 60*15); err != nil {
					log.Error(ctx, "cannot SetWithTTL: %s: %v", cacheKey, err)
				}
			}
		}
	}

	return pkey, pkey != ""
}
