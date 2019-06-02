package metricsservice

import (
	"github.com/ovh/cds/sdk"
)

func (s *Service) Status() sdk.MonitoringStatus {
	m := s.CommonMonitoring()

	for name, provider := range s.metricProviders {
		m.Lines = append(m.Lines, provider.GetStatus(name))
	}

	return m
}
