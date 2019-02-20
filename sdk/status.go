package sdk

import (
	"database/sql/driver"
	json "encoding/json"
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

// Value returns driver.Value from workflow template request.
func (s MonitoringStatus) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal MonitoringStatus")
}

// Scan workflow template request.
func (s *MonitoringStatus) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, s), "cannot unmarshal MonitoringStatus")
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
