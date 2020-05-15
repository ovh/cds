package cdn

import (
	"context"
	"net/http"
	"sync"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	onceMetrics        sync.Once
	Errors             *stats.Int64Measure
	Hits               *stats.Int64Measure
	WorkerLogReceived  *stats.Int64Measure
	ServiceLogReceived *stats.Int64Measure
)

func (s *Service) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(ctx), status)
	}
}

func (s *Service) Status(ctx context.Context) sdk.MonitoringStatus {
	m := s.CommonMonitoring()

	status := sdk.MonitoringStatusOK
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "CDN", Value: status, Status: status})
	return m
}

func (s *Service) InitMetrics() error {
	var err error
	onceMetrics.Do(func() {
		Errors = stats.Int64(
			"cdn/tcp/router_errors",
			"number of errors",
			stats.UnitDimensionless)
		Hits = stats.Int64(
			"cdn/tcp/router_hits",
			"number of hits",
			stats.UnitDimensionless)
		WorkerLogReceived = stats.Int64(
			"cdn/tcp/worker/log/count",
			"Number of worker log received",
			stats.UnitDimensionless)
		ServiceLogReceived = stats.Int64(
			"cdn/tcp/service/log/count",
			"Number of service log received",
			stats.UnitDimensionless)

		tagServiceType := observability.MustNewKey(observability.TagServiceType)
		tagServiceName := observability.MustNewKey(observability.TagServiceName)

		err = observability.RegisterView(
			observability.NewViewCount("cdn/tcp/router/router_errors", Errors, []tag.Key{tagServiceType, tagServiceName}),
			observability.NewViewCount("cdn/tcp/router/router_hits", Hits, []tag.Key{tagServiceType, tagServiceName}),
			observability.NewViewCount("cdn/tcp/worker/log/count", WorkerLogReceived, []tag.Key{tagServiceType, tagServiceName}),
			observability.NewViewCount("cdn/tcp/service/log/count", ServiceLogReceived, []tag.Key{tagServiceType, tagServiceName}),
		)
	})

	log.Debug("cdn> Stats initialized")

	return err
}
