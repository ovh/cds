package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type CanonicalService struct {
	ID         int64         `json:"id" db:"id"`
	Name       string        `json:"name" db:"name" cli:"name,key"`
	ConsumerID *string       `json:"-" db:"auth_consumer_id"`
	Type       string        `json:"type" db:"type" cli:"type"`
	HTTPURL    string        `json:"http_url" db:"http_url" cli:"url"`
	Config     ServiceConfig `json:"config" db:"config" cli:"-"`
	PublicKey  []byte        `json:"public_key" db:"public_key"`
}

// Service is a µService registered on CDS API.
type Service struct {
	CanonicalService
	LastHeartbeat    time.Time        `json:"last_heartbeat" db:"last_heartbeat" cli:"heartbeat"`
	MonitoringStatus MonitoringStatus `json:"monitoring_status" db:"monitoring_status" cli:"-"`
	Version          string           `json:"version" db:"-" cli:"version"`
	Uptodate         bool             `json:"up_to_date" db:"-"`
}

// Update service field from new data.
func (s *Service) Update(data Service) {
	s.Name = data.Name
	s.HTTPURL = data.HTTPURL
	s.Config = data.Config
	s.PublicKey = data.PublicKey
	s.LastHeartbeat = data.LastHeartbeat
	s.MonitoringStatus = data.MonitoringStatus
}

type ServiceConfig map[string]interface{}

// Value returns driver.Value from workflow template request.
func (c ServiceConfig) Value() (driver.Value, error) {
	j, err := json.Marshal(c)
	return j, WrapError(err, "cannot marshal ServiceConfig")
}

// Scan workflow template request.
func (c *ServiceConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, c), "cannot unmarshal ServiceConfig")
}

// ExternalService represents an external service
type ExternalService struct {
	Service    `json:"-"`
	HealthURL  string `json:"health_url"`
	HealthPort string `json:"health_port"`
	HealthPath string `json:"health_path"`
	Port       string `json:"port"`
	URL        string `json:"url"`
	Path       string `json:"path"`
}

func (e ExternalService) ServiceConfig() ServiceConfig {
	b, _ := json.Marshal(e)
	var cfg ServiceConfig
	json.Unmarshal(b, &cfg) // nolint
	return cfg
}
