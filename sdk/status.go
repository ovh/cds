package sdk

import (
	"fmt"
	"net/http"
	"time"
)

// This constants deals with Monitoring statuses
const (
	MonitoringStatusAlert = "AL"
	MonitoringStatusWarn  = "WARN"
	MonitoringStatusOK    = "OK"
)

// MonitoringStatus contains status of CDS Component
type MonitoringStatus struct {
	Now   time.Time              `json:"now"`
	Lines []MonitoringStatusLine `json:"lines"`
}

// MonitoringStatusLine represents a CDS Component Status
type MonitoringStatusLine struct {
	Status    string `json:"status" cli:"status"`
	Component string `json:"component" cli:"component"`
	Value     string `json:"value" cli:"value"`
	Type      string `json:"type" cli:"type"`
}

// HTTPStatusCode return the http status code
func (m MonitoringStatus) HTTPStatusCode() int {
	for _, l := range m.Lines {
		if l.Status != MonitoringStatusOK {
			return http.StatusServiceUnavailable
		}
	}
	return http.StatusOK
}

func (m MonitoringStatusLine) String() string {
	return fmt.Sprintf("%s - %s: %s", m.Status, m.Component, m.Value)
}
