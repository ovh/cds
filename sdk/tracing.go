package sdk

import (
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

// B3 headers that OpenCensus understands.
const (
	TraceIDHeader = "X-B3-TraceId"
	SpanIDHeader  = "X-B3-SpanId"
	SampledHeader = "X-B3-Sampled"
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
