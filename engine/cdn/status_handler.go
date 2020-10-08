package cdn

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
func (s *Service) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := s.NewMonitoringStatus()

	if !s.Cfg.EnableLogProcessing {
		return m
	}
	db := s.mustDBWithCtx(ctx)

	nbCompleted, err := storage.CountItemCompleted(db)
	m.AddLine(addMonitoringLine(nbCompleted, "items/completed", err, sdk.MonitoringStatusOK))

	nbIncoming, err := storage.CountItemIncoming(db)
	m.AddLine(addMonitoringLine(nbIncoming, "items/incoming", err, sdk.MonitoringStatusOK))

	m.AddLine(s.LogCache.Status(ctx)...)
	m.AddLine(s.getStatusSyncLogs()...)

	for _, st := range s.Units.Storages {
		m.AddLine(s.computeStatusBackend(ctx, db, nbCompleted, st)...)
	}

	m.AddLine(s.DBConnectionFactory.Status(ctx))

	return m
}

func (s *Service) computeStatusBackend(ctx context.Context, db *gorp.DbMap, nbCompleted int64, storageUnit storage.StorageUnit) []sdk.MonitoringStatusLine {
	lines := storageUnit.Status(ctx)

	currentSize, err := storage.CountItemUnitByUnit(db, storageUnit.ID())
	if err != nil {
		log.Info(ctx, "cdn:status: err:%v", err)
		lines = append(lines, addMonitoringLine(currentSize, "backend/"+storageUnit.Name()+"/items", err, sdk.MonitoringStatusAlert))
	} else {
		lines = append(lines, addMonitoringLine(currentSize, "backend/"+storageUnit.Name()+"/items", err, sdk.MonitoringStatusOK))
	}

	var previousLag, previousSize int64

	lagKey := storageUnit.ID() + "lag"
	sizeKey := storageUnit.ID() + "size"

	// load previous values computed
	r, ok := s.storageUnitLags.Load(lagKey)
	if !ok {
		previousLag = 0
	} else {
		previousLag = r.(int64)
	}
	siz, ok := s.storageUnitLags.Load(sizeKey)
	if !ok {
		previousSize = 0
	} else {
		previousSize = siz.(int64)
	}

	currentLag := nbCompleted - currentSize
	// if we have less lag than previous compute or if the currentSize is greater than previous compute, it's OK
	if currentLag == 0 || (currentLag > 0 && currentLag < previousLag || currentSize > previousSize) {
		lines = append(lines, addMonitoringLine(currentLag, "backend/"+storageUnit.Name()+"/lag", err, sdk.MonitoringStatusOK))
	} else {
		lines = append(lines, addMonitoringLine(currentLag, "backend/"+storageUnit.Name()+"/lag", err, sdk.MonitoringStatusWarn))
	}

	s.storageUnitLags.Store(lagKey, currentLag)
	s.storageUnitLags.Store(sizeKey, currentSize)
	return lines
}
