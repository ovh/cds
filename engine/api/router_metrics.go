package api

// A part of this file is coming from https://github.com/census-instrumentation/opencensus-go/blob/master/plugin/ochttp/server.go

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

import (
	"io"
	"net/http"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/log"
)

// InitRouterMetrics initialize prometheus metrics
func InitRouterMetrics(s service.NamedService) error {
	var err error
	onceMetrics.Do(func() {
		Errors = stats.Int64(
			"cds/router_errors",
			"number of errors",
			stats.UnitDimensionless)
		Hits = stats.Int64(
			"cds/router_hits",
			"number of hits",
			stats.UnitDimensionless)
		SSEClients = stats.Int64(
			"cds/sse_clients",
			"number of sse clients",
			stats.UnitDimensionless)
		SSEEvents = stats.Int64(
			"cds/sse_events",
			"number of sse events",
			stats.UnitDimensionless)
		WebSocketClients = stats.Int64(
			"cds/websocket_clients",
			"number of  websocket clients",
			stats.UnitDimensionless)
		WebSocketEvents = stats.Int64(
			"cds/websocket_events",
			"number of websocket events",
			stats.UnitDimensionless)
		ServerRequestCount = stats.Int64(
			"cds/http/server/request_count",
			"Number of HTTP requests started",
			stats.UnitDimensionless)
		ServerRequestBytes = stats.Int64(
			"cds/http/server/request_bytes",
			"HTTP request body size if set as ContentLength (uncompressed)",
			stats.UnitBytes)
		ServerResponseBytes = stats.Int64(
			"cds/http/server/response_bytes",
			"HTTP response body size (uncompressed)",
			stats.UnitBytes)
		ServerLatency = stats.Float64(
			"cds/http/server/latency",
			"End-to-end latency",
			stats.UnitMilliseconds)

		tagServiceType := observability.MustNewKey(observability.TagServiceType)
		tagServiceName := observability.MustNewKey(observability.TagServiceName)

		ServerRequestCountView := &view.View{
			Name:        "cds/http/server/request_count_by_handler",
			Description: "Count of HTTP requests started",
			Measure:     ServerRequestCount,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, observability.MustNewKey(observability.Handler)},
			Aggregation: view.Count(),
		}

		ServerRequestBytesView := &view.View{
			Name:        "cds/http/server/request_bytes_by_handler",
			Description: "Size distribution of HTTP request body",
			Measure:     ServerRequestBytes,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, observability.MustNewKey(observability.Handler)},
			Aggregation: observability.DefaultSizeDistribution,
		}

		ServerResponseBytesView := &view.View{
			Name:        "cds/http/server/response_bytes_by_handler",
			Description: "Size distribution of HTTP response body",
			Measure:     ServerResponseBytes,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, observability.MustNewKey(observability.Handler)},
			Aggregation: observability.DefaultSizeDistribution,
		}

		ServerLatencyView := &view.View{
			Name:        "cds/http/server/latency_by_handler",
			Description: "Latency distribution of HTTP requests",
			Measure:     ServerLatency,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, observability.MustNewKey(observability.Handler)},
			Aggregation: observability.DefaultLatencyDistribution,
		}

		ServerRequestCountByMethod := &view.View{
			Name:        "cds/http/server/request_count_by_method_and_handler",
			Description: "Server request count by HTTP method",
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, observability.MustNewKey(observability.Method), observability.MustNewKey(observability.Handler)},
			Measure:     ServerRequestCount,
			Aggregation: view.Count(),
		}

		ServerResponseCountByStatusCode := &view.View{
			Name:        "cds/http/server/response_count_by_status_code_and_handler",
			Description: "Server response count by status code",
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, observability.MustNewKey(observability.StatusCode), observability.MustNewKey(observability.Handler)},
			Measure:     ServerLatency,
			Aggregation: view.Count(),
		}

		err = observability.RegisterView(
			observability.NewViewCount("cds/http/router/router_errors", Errors, []tag.Key{tagServiceType, tagServiceName}),
			observability.NewViewCount("cds/http/router/router_hits", Hits, []tag.Key{tagServiceType, tagServiceName}),
			observability.NewViewLast("cds/http/router/sse_clients", SSEClients, []tag.Key{tagServiceType, tagServiceName}),
			observability.NewViewCount("cds/http/router/sse_events", SSEEvents, []tag.Key{tagServiceType, tagServiceName}),
			observability.NewViewLast("cds/http/router/websocket_clients", WebSocketClients, []tag.Key{tagServiceType, tagServiceName}),
			observability.NewViewCount("cds/http/router/websocket_events", WebSocketEvents, []tag.Key{tagServiceType, tagServiceName}),
			ServerRequestCountView,
			ServerRequestBytesView,
			ServerResponseBytesView,
			ServerLatencyView,
			ServerRequestCountByMethod,
			ServerResponseCountByStatusCode,
		)
	})

	log.Debug("router> Stats initialized")

	return err
}

type trackingResponseWriter struct {
	statusCode int
	statusLine string
	writer     http.ResponseWriter
	reqSize    int64
	respSize   int64
}

// Compile time assertion for ResponseWriter interface
var _ http.ResponseWriter = (*trackingResponseWriter)(nil)

func (t *trackingResponseWriter) Header() http.Header {
	return t.writer.Header()
}

func (t *trackingResponseWriter) Write(data []byte) (int, error) {
	n, err := t.writer.Write(data)
	t.respSize += int64(n)
	return n, err
}

func (t *trackingResponseWriter) WriteHeader(statusCode int) {
	t.writer.WriteHeader(statusCode)
	t.statusCode = statusCode
	t.statusLine = http.StatusText(t.statusCode)
}

// wrappedResponseWriter returns a wrapped version of the original
//  ResponseWriter and only implements the same combination of additional
// interfaces as the original.
// This implementation is based on https://github.com/felixge/httpsnoop.
func (t *trackingResponseWriter) wrappedResponseWriter() http.ResponseWriter {
	var (
		hj, i0 = t.writer.(http.Hijacker)
		cn, i1 = t.writer.(http.CloseNotifier)
		pu, i2 = t.writer.(http.Pusher)
		fl, i3 = t.writer.(http.Flusher)
		rf, i4 = t.writer.(io.ReaderFrom)
	)

	switch {
	case !i0 && !i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
		}{t}
	case !i0 && !i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			io.ReaderFrom
		}{t, rf}
	case !i0 && !i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Flusher
		}{t, fl}
	case !i0 && !i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
		}{t, fl, rf}
	case !i0 && !i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Pusher
		}{t, pu}
	case !i0 && !i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			io.ReaderFrom
		}{t, pu, rf}
	case !i0 && !i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			http.Flusher
		}{t, pu, fl}
	case !i0 && !i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, pu, fl, rf}
	case !i0 && i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
		}{t, cn}
	case !i0 && i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			io.ReaderFrom
		}{t, cn, rf}
	case !i0 && i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
		}{t, cn, fl}
	case !i0 && i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			io.ReaderFrom
		}{t, cn, fl, rf}
	case !i0 && i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
		}{t, cn, pu}
	case !i0 && i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			io.ReaderFrom
		}{t, cn, pu, rf}
	case !i0 && i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			http.Flusher
		}{t, cn, pu, fl}
	case !i0 && i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, cn, pu, fl, rf}
	case i0 && !i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
		}{t, hj}
	case i0 && !i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
		}{t, hj, rf}
	case i0 && !i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Flusher
		}{t, hj, fl}
	case i0 && !i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Flusher
			io.ReaderFrom
		}{t, hj, fl, rf}
	case i0 && !i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
		}{t, hj, pu}
	case i0 && !i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{t, hj, pu, rf}
	case i0 && !i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			http.Flusher
		}{t, hj, pu, fl}
	case i0 && !i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, hj, pu, fl, rf}
	case i0 && i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
		}{t, hj, cn}
	case i0 && i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			io.ReaderFrom
		}{t, hj, cn, rf}
	case i0 && i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Flusher
		}{t, hj, cn, fl}
	case i0 && i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Flusher
			io.ReaderFrom
		}{t, hj, cn, fl, rf}
	case i0 && i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
		}{t, hj, cn, pu}
	case i0 && i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
			io.ReaderFrom
		}{t, hj, cn, pu, rf}
	case i0 && i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
			http.Flusher
		}{t, hj, cn, pu, fl}
	case i0 && i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, hj, cn, pu, fl, rf}
	default:
		return struct {
			http.ResponseWriter
		}{t}
	}
}
