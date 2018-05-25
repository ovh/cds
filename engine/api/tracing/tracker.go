package tracing

import (
	"encoding/hex"
	"net/http"

	"github.com/ovh/cds/sdk"
	"go.opencensus.io/trace"
)

func spanContextToReponse(ctx trace.SpanContext, r *http.Request, w http.ResponseWriter) {
	w.Header().Add(sdk.TraceIDHeader, hex.EncodeToString(ctx.TraceID[:]))
	w.Header().Add(sdk.SpanIDHeader, hex.EncodeToString(ctx.SpanID[:]))
	w.Header().Add(sdk.SampledHeader, r.Header.Get(sdk.SampledHeader))
}

// spanContextToRequest writes span context to http requests
func spanContextToRequest(ctx trace.SpanContext, r *http.Request) {
	defaultFormat.SpanContextToRequest(ctx, r)
}
