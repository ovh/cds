package elasticsearch

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := s.NewMonitoringStatus()
	var value, status string
	if s.esClient == nil {
		status = sdk.MonitoringStatusWarn
		value = "disconnected"
	} else {
		success, err := s.esClient.Ping(ctx)
		if err != nil {
			status = sdk.MonitoringStatusWarn
			value = fmt.Sprintf("no ping (%v)", err)
		} else if !success {
			status = sdk.MonitoringStatusWarn
			value = fmt.Sprintf("ping error")
		} else {
			status = sdk.MonitoringStatusOK
		}
	}
	m.AddLine(sdk.MonitoringStatusLine{Component: "Elasticsearch", Value: value, Status: status})
	return m
}
