package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

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
	Status    string `json:"status"`
	Component string `json:"component"`
	Value     string `json:"value"`
}

func (m MonitoringStatusLine) String() string {
	return fmt.Sprintf("%s - %s: %s", m.Status, m.Component, m.Value)
}

// GetStatus retrieve generic health infos to CDS
func GetStatus() (*MonitoringStatus, error) {
	output := MonitoringStatus{}
	data, code, err := Request("GET", "/mon/status", nil)
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

// GetVersion returns API version
func GetVersion() (string, error) {
	data, code, err := Request("GET", "/mon/version", nil)
	if err != nil {
		return "", err
	}

	if code >= 300 {
		return "", fmt.Errorf("HTTP %d", code)
	}

	s := struct {
		Version string `json:"version"`
	}{}

	if err := json.Unmarshal(data, &s); err != nil {
		return "", err
	}

	return s.Version, nil
}
