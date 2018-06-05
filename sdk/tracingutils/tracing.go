package tracingutils

import (
	"context"
	"encoding/hex"

	"go.opencensus.io/trace"
)

// This is a copy and paste of https://github.com/census-instrumentation/opencensus-go/blob/3a827557227b08de330abfac83b8299810c42ac2/plugin/ochttp/propagation/b3/b3.go
// Waiting it to be released
//
// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

type contextKey int

// B3 headers that OpenCensus understands.
const (
	TraceIDHeader = "X-B3-TraceId"
	SpanIDHeader  = "X-B3-SpanId"
	SampledHeader = "X-B3-Sampled"

	ContextTraceIDHeader contextKey = iota
	ContextSpanIDHeader
	ContextSampledHeader
)

// ParseTraceID parses the value of the X-B3-TraceId header.
func ParseTraceID(tid string) (trace.TraceID, bool) {
	if tid == "" {
		return trace.TraceID{}, false
	}
	b, err := hex.DecodeString(tid)
	if err != nil {
		return trace.TraceID{}, false
	}
	var traceID trace.TraceID
	if len(b) <= 8 {
		// The lower 64-bits.
		start := 8 + (8 - len(b))
		copy(traceID[start:], b)
	} else {
		start := 16 - len(b)
		copy(traceID[start:], b)
	}

	return traceID, true
}

// ParseSpanID parses the value of the X-B3-SpanId or X-B3-ParentSpanId headers.
func ParseSpanID(sid string) (spanID trace.SpanID, ok bool) {
	if sid == "" {
		return trace.SpanID{}, false
	}
	b, err := hex.DecodeString(sid)
	if err != nil {
		return trace.SpanID{}, false
	}
	start := 8 - len(b)
	copy(spanID[start:], b)
	return spanID, true
}

// ParseSampled parses the value of the X-B3-Sampled header.
func ParseSampled(sampled string) (trace.TraceOptions, bool) {
	switch sampled {
	case "true", "1":
		return trace.TraceOptions(1), true
	default:
		return trace.TraceOptions(0), false
	}
}

// SpanContextToContext merge a span context in a context
func SpanContextToContext(ctx context.Context, sc trace.SpanContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx = context.WithValue(ctx, ContextTraceIDHeader, sc.TraceID)
	ctx = context.WithValue(ctx, ContextSpanIDHeader, sc.SpanID)
	ctx = context.WithValue(ctx, ContextSampledHeader, sc.IsSampled())
	return ctx
}

// ContextToSpanContext instanciates a span context from a context.Context
func ContextToSpanContext(ctx context.Context) (trace.SpanContext, bool) {
	if ctx == nil {
		return trace.SpanContext{}, false
	}

	val := ctx.Value(ContextTraceIDHeader)
	if val == nil {
		return trace.SpanContext{}, false
	}
	traceID, ok := val.(trace.TraceID)
	if !ok {
		return trace.SpanContext{}, false
	}

	val = ctx.Value(ContextSpanIDHeader)
	if val == nil {
		return trace.SpanContext{}, false
	}
	spanID, ok := val.(trace.SpanID)
	if !ok {
		return trace.SpanContext{}, false
	}

	val = ctx.Value(ContextSpanIDHeader)
	if val == nil {
		return trace.SpanContext{}, false
	}
	sampled, ok := val.(trace.TraceOptions)
	if !ok {
		return trace.SpanContext{}, false
	}

	sc := trace.SpanContext{
		TraceID:      traceID,
		SpanID:       spanID,
		TraceOptions: sampled,
	}

	return sc, true
}
