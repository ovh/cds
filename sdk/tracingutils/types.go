package tracingutils

import (
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/trace/propagation"
)

// DefaultFormat used by observability as: observability.DefaultFormat.SpanContextToRequest
var DefaultFormat propagation.HTTPFormat = &b3.HTTPFormat{}
