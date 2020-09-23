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
	if esClient == nil {
		status = sdk.MonitoringStatusWarn
		value = "disconnected"
	} else {
		_, code, err := esClient.Ping(s.Cfg.ElasticSearch.URL).Do(context.Background())
		if err != nil {
			status = sdk.MonitoringStatusWarn
			value = fmt.Sprintf("no ping (%v)", err)
		} else if code >= 400 {
			status = sdk.MonitoringStatusWarn
			value = fmt.Sprintf("ping error (code:%d, err: %v)", code, err)
		} else {
			status = sdk.MonitoringStatusOK
		}
	}
	m.AddLine(sdk.MonitoringStatusLine{Component: "Elasticsearch", Value: value, Status: status})
	return m
}
