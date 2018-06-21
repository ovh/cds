package elasticsearch

import (
	"context"

	"github.com/ovh/cds/sdk"
)

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status() sdk.MonitoringStatus {
	m := s.CommonMonitoring()
	var value, status string
	if esClient == nil {
		status = sdk.MonitoringStatusWarn
		value = "disconnected"
	} else {
		_, code, err := esClient.Ping(s.Cfg.URL).Do(context.Background())
		if err != nil {
			status = sdk.MonitoringStatusWarn
			value = "no ping"
		} else if code >= 400 {
			status = sdk.MonitoringStatusWarn
			value = "ping error"
		} else {
			status = sdk.MonitoringStatusOK
		}
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Elasticsearch", Value: value, Status: status})
	return m
}
