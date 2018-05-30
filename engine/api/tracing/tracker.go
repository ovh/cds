package tracing

import (
	"encoding/hex"
	"net/http"

	"go.opencensus.io/trace"

	"github.com/ovh/cds/sdk/tracingutils"
)

func spanContextToReponse(ctx trace.SpanContext, r *http.Request, w http.ResponseWriter) {
	w.Header().Add(tracingutils.TraceIDHeader, hex.EncodeToString(ctx.TraceID[:]))
	w.Header().Add(tracingutils.SpanIDHeader, hex.EncodeToString(ctx.SpanID[:]))
	w.Header().Add(tracingutils.SampledHeader, r.Header.Get(tracingutils.SampledHeader))
}
