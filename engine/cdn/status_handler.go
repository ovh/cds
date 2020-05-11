package cdn

import (
	"context"
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

func (s *Service) Status(ctx context.Context) sdk.MonitoringStatus {
	m := s.CommonMonitoring()

	status := sdk.MonitoringStatusOK
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "CDN", Value: status, Status: status})
	return m
}
