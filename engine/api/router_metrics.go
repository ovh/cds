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
	"context"

	"github.com/rockbears/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/telemetry"
)

// InitRouterMetrics initialize prometheus metrics
func InitRouterMetrics(ctx context.Context, s service.NamedService) error {
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
		WebSocketClients = stats.Int64(
			"cds/websocket_clients",
			"number of websocket clients",
			stats.UnitDimensionless)
		WebSocketV2Clients = stats.Int64(
			"cds/websocket_v2_clients",
			"number of websocket v2 clients",
			stats.UnitDimensionless)
		WebSocketEvents = stats.Int64(
			"cds/websocket_events",
			"number of websocket events",
			stats.UnitDimensionless)
		WebSocketV2Events = stats.Int64(
			"cds/websocket_v2_events",
			"number of websocket v2 events",
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

		tagServiceType := telemetry.MustNewKey(telemetry.TagServiceType)
		tagServiceName := telemetry.MustNewKey(telemetry.TagServiceName)

		ServerRequestCountView := &view.View{
			Name:        "cds/http/server/request_count_by_handler",
			Description: "Count of HTTP requests started",
			Measure:     ServerRequestCount,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, telemetry.MustNewKey(telemetry.Handler)},
			Aggregation: view.Count(),
		}

		ServerRequestBytesView := &view.View{
			Name:        "cds/http/server/request_bytes_by_handler",
			Description: "Size distribution of HTTP request body",
			Measure:     ServerRequestBytes,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, telemetry.MustNewKey(telemetry.Handler)},
			Aggregation: telemetry.DefaultSizeDistribution,
		}

		ServerResponseBytesView := &view.View{
			Name:        "cds/http/server/response_bytes_by_handler",
			Description: "Size distribution of HTTP response body",
			Measure:     ServerResponseBytes,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, telemetry.MustNewKey(telemetry.Handler)},
			Aggregation: telemetry.DefaultSizeDistribution,
		}

		ServerLatencyView := &view.View{
			Name:        "cds/http/server/latency_by_handler",
			Description: "Latency distribution of HTTP requests",
			Measure:     ServerLatency,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, telemetry.MustNewKey(telemetry.Handler)},
			Aggregation: telemetry.DefaultLatencyDistribution,
		}

		ServerRequestCountByMethod := &view.View{
			Name:        "cds/http/server/request_count_by_method_and_handler",
			Description: "Server request count by HTTP method",
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, telemetry.MustNewKey(telemetry.Method), telemetry.MustNewKey(telemetry.Handler)},
			Measure:     ServerRequestCount,
			Aggregation: view.Count(),
		}

		ServerResponseCountByStatusCode := &view.View{
			Name:        "cds/http/server/response_count_by_status_code_and_handler",
			Description: "Server response count by status code",
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, telemetry.MustNewKey(telemetry.StatusCode), telemetry.MustNewKey(telemetry.Handler)},
			Measure:     ServerLatency,
			Aggregation: view.Count(),
		}

		err = telemetry.RegisterView(ctx,
			telemetry.NewViewCount("cds/http/router/router_errors", Errors, []tag.Key{tagServiceType, tagServiceName}),
			telemetry.NewViewCount("cds/http/router/router_hits", Hits, []tag.Key{tagServiceType, tagServiceName}),
			telemetry.NewViewLast("cds/http/router/websocket_clients", WebSocketClients, []tag.Key{tagServiceType, tagServiceName}),
			telemetry.NewViewLast("cds/http/router/websocket_v2_clients", WebSocketV2Clients, []tag.Key{tagServiceType, tagServiceName}),
			telemetry.NewViewCount("cds/http/router/websocket_events", WebSocketEvents, []tag.Key{tagServiceType, tagServiceName}),
			telemetry.NewViewCount("cds/http/router/websocket_v2_events", WebSocketV2Events, []tag.Key{tagServiceType, tagServiceName}),
			ServerRequestCountView,
			ServerRequestBytesView,
			ServerResponseBytesView,
			ServerLatencyView,
			ServerRequestCountByMethod,
			ServerResponseCountByStatusCode,
		)
	})

	log.Debug(ctx, "router> Stats initialized")

	return err
}
