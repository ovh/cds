package cdn

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
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

func addMonitoringLine(nb int64, text string, err error, status string) sdk.MonitoringStatusLine {
	if err != nil {
		return sdk.MonitoringStatusLine{
			Component: text,
			Value:     fmt.Sprintf("Error: %v", err),
			Status:    sdk.MonitoringStatusAlert,
		}
	} else {
		return sdk.MonitoringStatusLine{
			Component: text,
			Value:     fmt.Sprintf("%d", nb),
			Status:    status,
		}
	}
}

func (s *Service) Status(ctx context.Context) sdk.MonitoringStatus {
	m := s.CommonMonitoring()
	db := s.mustDBWithCtx(ctx)

	nbCompleted, err := storage.CountItemCompleted(db)
	m.Lines = append(m.Lines, addMonitoringLine(nbCompleted, "index/items/completed", err, sdk.MonitoringStatusOK))

	nbIncoming, err := storage.CountItemIncoming(db)
	m.Lines = append(m.Lines, addMonitoringLine(nbIncoming, "index/items/incoming", err, sdk.MonitoringStatusOK))

	for _, st := range s.Units.Storages {
		m.Lines = append(m.Lines, st.Status()...)
		size, err := storage.CountItemUnitByUnit(db, st.ID())
		if nbCompleted-size >= 100 {
			m.Lines = append(m.Lines, addMonitoringLine(size, "backend/"+st.Name()+"/index/items", err, sdk.MonitoringStatusWarn))
		} else {
			m.Lines = append(m.Lines, addMonitoringLine(size, "backend/"+st.Name()+"/index/items", err, sdk.MonitoringStatusOK))
		}
	}

	m.Lines = append(m.Lines, s.DBConnectionFactory.Status(ctx))

	return m
}

func (s *Service) initMetrics(ctx context.Context) error {
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

		tagServiceType := telemetry.MustNewKey(telemetry.TagServiceType)
		tagServiceName := telemetry.MustNewKey(telemetry.TagServiceName)

		err = telemetry.RegisterView(ctx,
			telemetry.NewViewCount("cdn/tcp/router/router_errors", Errors, []tag.Key{tagServiceType, tagServiceName}),
			telemetry.NewViewCount("cdn/tcp/router/router_hits", Hits, []tag.Key{tagServiceType, tagServiceName}),
			telemetry.NewViewCount("cdn/tcp/worker/log/count", WorkerLogReceived, []tag.Key{tagServiceType, tagServiceName}),
			telemetry.NewViewCount("cdn/tcp/service/log/count", ServiceLogReceived, []tag.Key{tagServiceType, tagServiceName}),
		)
	})

	log.Debug("cdn> Stats initialized")

	return err
}
