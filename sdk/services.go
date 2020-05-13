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
	LogServer        TCPServer        `json:"tcp" db:"-"`
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

// ServiceConfiguration is the configuration of service
type ServiceConfiguration struct {
	Name       string `toml:"name" json:"name"`
	URL        string `toml:"url" json:"url"`
	Port       string `toml:"port" json:"port"`
	Path       string `toml:"path" json:"path"`
	HealthURL  string `toml:"healthUrl" json:"health_url"`
	HealthPort string `toml:"healthPort" json:"health_port"`
	HealthPath string `toml:"healthPath" json:"health_path"`
	Type       string `toml:"type" json:"type"`
	PublicKey  string `json:"publicKey"`
	ID         int64  `json:"id"`
}
