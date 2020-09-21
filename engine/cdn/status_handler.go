package cdn

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

var (
	onceMetrics               sync.Once
	metricsErrors             *stats.Int64Measure
	metricsHits               *stats.Int64Measure
	metricsStepLogReceived    *stats.Int64Measure
	metricsServiceLogReceived *stats.Int64Measure
	metricsItemCompletedByGC  *stats.Int64Measure
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
	}
	return sdk.MonitoringStatusLine{
		Component: text,
		Value:     fmt.Sprintf("%d", nb),
		Status:    status,
	}
}

// Status returns the monitoring status for this service
func (s *Service) Status(ctx context.Context) sdk.MonitoringStatus {
	m := s.CommonMonitoring()

	if !s.Cfg.EnableLogProcessing {
		return m
	}
	db := s.mustDBWithCtx(ctx)

	nbCompleted, err := storage.CountItemCompleted(db)
	m.Lines = append(m.Lines, addMonitoringLine(nbCompleted, "items/completed", err, sdk.MonitoringStatusOK))

	nbIncoming, err := storage.CountItemIncoming(db)
	m.Lines = append(m.Lines, addMonitoringLine(nbIncoming, "items/incoming", err, sdk.MonitoringStatusOK))

	m.Lines = append(m.Lines, s.LogCache.Status(ctx)...)
	m.Lines = append(m.Lines, s.getStatusSyncLogs()...)

	for _, st := range s.Units.Storages {
		m.Lines = append(m.Lines, st.Status(ctx)...)
		size, err := storage.CountItemUnitByUnit(db, st.ID())
		if nbCompleted-size >= 100 {
			m.Lines = append(m.Lines, addMonitoringLine(size, "backend/"+st.Name()+"/items", err, sdk.MonitoringStatusWarn))
		} else {
			m.Lines = append(m.Lines, addMonitoringLine(size, "backend/"+st.Name()+"/items", err, sdk.MonitoringStatusOK))
		}
	}

	m.Lines = append(m.Lines, s.DBConnectionFactory.Status(ctx))

	return m
}

func (s *Service) initMetrics(ctx context.Context) error {
	var err error
	onceMetrics.Do(func() {
		metricsErrors = stats.Int64("cdn/tcp/router_errors", "number of errors", stats.UnitDimensionless)
		metricsHits = stats.Int64("cdn/tcp/router_hits", "number of hits", stats.UnitDimensionless)
		metricsStepLogReceived = stats.Int64("cdn/tcp/step/log/count", "number of worker log received", stats.UnitDimensionless)
		metricsServiceLogReceived = stats.Int64("cdn/tcp/service/log/count", "number of service log received", stats.UnitDimensionless)
		metricsItemCompletedByGC = stats.Int64("cdn/items/completed_by_gc", "number of items completed by GC", stats.UnitDimensionless)

		err = telemetry.InitMetricsInt64(ctx, metricsErrors, metricsHits, metricsServiceLogReceived, metricsServiceLogReceived, metricsItemCompletedByGC)
	})

	log.Debug("cdn> Stats initialized")

	return err
}
