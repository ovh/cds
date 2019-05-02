package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"fmt"
	"time"
)

// Service is a µService registered on CDS API
type Service struct {
	ID               int64            `json:"id" db:"id"`
	Name             string           `json:"name" db:"name" cli:"name,key"`
	Type             string           `json:"type" db:"type" cli:"type"`
	HTTPURL          string           `json:"http_url" db:"http_url" cli:"url"`
	LastHeartbeat    time.Time        `json:"last_heartbeat" db:"last_heartbeat" cli:"heartbeat"`
	Hash             string           `json:"hash" db:"hash"`
	Token            string           `json:"token" db:"-"`
	GroupID          *int64           `json:"group_id" db:"group_id"`
	Group            *Group           `json:"group" db:"-"`
	MonitoringStatus MonitoringStatus `json:"monitoring_status" db:"monitoring_status" cli:"-"`
	Config           ServiceConfig    `json:"config" db:"config" cli:"-"`
	IsSharedInfra    bool             `json:"is_shared_infra" db:"-"`
	Version          string           `json:"version" db:"-" cli:"version"`
	Uptodate         bool             `json:"up_to_date" db:"-"`
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
	json.Unmarshal(b, cfg) // nolint
	return cfg
}
