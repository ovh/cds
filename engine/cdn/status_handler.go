package cdn

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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

	m.AddLine(s.LogCache.Status(ctx)...)

	for _, st := range s.Units.Storages {
		m.AddLine(st.Status(ctx)...)
	}

	s.storageUnitLags.Range(func(key, cl interface{}) bool {
		currentLag := cl.(int64)

		pl, ok := s.storageUnitLags.Load(key)
		if !ok {
			return true
		}
		previousLag := pl.(int64)

		ps, ok := s.storageUnitPreviousSizes.Load(key)
		if !ok {
			return true
		}
		previousSize := ps.(int64)

		cs, ok := s.storageUnitSizes.Load(key)
		if !ok {
			return true
		}
		currentSize := cs.(int64)

		// if we have less lag than previous compute or if the currentSize is greater than previous compute, it's OK
		if currentLag == 0 || (currentLag > 0 && currentLag < previousLag || currentSize > previousSize) {
			m.AddLine(addMonitoringLine(currentLag, key.(string)+"/lag", nil, sdk.MonitoringStatusOK))
		} else {
			m.AddLine(addMonitoringLine(currentLag, key.(string)+"/lag", nil, sdk.MonitoringStatusWarn))
		}
		return true
	})

	m.AddLine(s.DBConnectionFactory.Status(ctx))

	return m
}
