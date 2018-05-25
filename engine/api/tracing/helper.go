package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
	"go.opencensus.io/trace"
)

// LinkTo a traceID
func LinkTo(ctx context.Context, traceID [16]byte) {
	s := Current(ctx)
	if s == nil {
		return
	}

	s.AddLink(
		trace.Link{
			TraceID: trace.TraceID(traceID),
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

// Tag is helper function to instanciate trace.Attribute
func Tag(key string, value interface{}) trace.Attribute {
	return trace.StringAttribute(key, fmt.Sprintf("%v", value))
}

// Span start a new span from the parent context
func Span(ctx context.Context, name string, tags ...trace.Attribute) (context.Context, func()) {
	if ctx == nil {
		return nil, func() {}
	}
	var span *trace.Span
	ctx, span = trace.StartSpan(ctx, name)
	if len(tags) > 0 {
		span.AddAttributes(tags...)
	}
	return ctx, span.End
}

func findPrimaryKeyFromRequest(req *http.Request, db gorp.SqlExecutor, store cache.Store) (string, bool) {
	vars := mux.Vars(req)
	pkey := vars["key"]
	if pkey == "" {
		pkey = vars["permProjectKey"]
	}

	if pkey == "" {
		id, _ := strconv.ParseInt(vars["id"], 10, 64)
		//The ID found may be a node run job, let's try to find the project key behing
		if id <= 0 {
			id, _ = strconv.ParseInt(vars["permID"], 10, 64)
		}
		if id != 0 {
			var err error
			cacheKey := cache.Key("api:FindProjetKeyForNodeRunJob:", fmt.Sprintf("%v", id))
			if !store.Get(cacheKey, &pkey) {
				pkey, err = findProjetKeyForNodeRunJob(db, id)
				if err != nil {
					log.Error("tracingMiddleware> %v", err)
					return "", false
				}
				store.SetWithTTL(cacheKey, pkey, 60*15)
			}
		}
	}

	return pkey, pkey != ""
}
