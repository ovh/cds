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

	ServiceType     string `json:"-"`
	ServiceName     string `json:"-"`
	ServiceHostname string `json:"-"`
}

// Value returns driver.Value from workflow template request.
func (m MonitoringStatus) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal MonitoringStatus")
}

// Scan workflow template request.
func (m *MonitoringStatus) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, m), "cannot unmarshal MonitoringStatus")
}

// AddLine adds line to MonitoringStatus, including the Type of component
func (m *MonitoringStatus) AddLine(lines ...MonitoringStatusLine) {
	for i := range lines {
		l := lines[i]
		l.Type = m.ServiceType
		l.Service = m.ServiceName
		l.Hostname = m.ServiceHostname
		m.Lines = append(m.Lines, l)
	}
}

// MonitoringStatusLine represents a CDS Component Status
type MonitoringStatusLine struct {
	Status     string `json:"status" cli:"status"`
	Component  string `json:"component" cli:"component"`
	Value      string `json:"value" cli:"value"`
	Type       string `json:"type" cli:"type"`
	Service    string `json:"service" cli:"service"`
	Hostname   string `json:"hostname" cli:"hostname"`
	SessionID  string `json:"session,omitempty" cli:"session"`
	ConsumerID string `json:"consumer,omitempty" cli:"consumer"`
}

// HTTPStatusCode returns the http status code
func (m MonitoringStatus) HTTPStatusCode() int {
	for _, l := range m.Lines {
		if l.Status == MonitoringStatusAlert {
			return http.StatusServiceUnavailable
		}
	}
	return http.StatusOK
}

func (l MonitoringStatusLine) String() string {
	return fmt.Sprintf("%s - %s: %s", l.Status, l.Component, l.Value)
}
