package sdk

import (
	"time"
)

// Service is a µService registered on CDS API
type Service struct {
	Name             string           `json:"name" db:"name" cli:"name,key"`
	Type             string           `json:"type" db:"type" cli:"type"`
	HTTPURL          string           `json:"http_url" db:"http_url" cli:"url"`
	LastHeartbeat    time.Time        `json:"last_heartbeat" db:"last_heartbeat" cli:"heartbeat"`
	Hash             string           `json:"hash" db:"hash"`
	Token            string           `json:"token" db:"-"`
	MonitoringStatus MonitoringStatus `json:"monitoring_status" db:"-" cli:"-"`
}

// ExternalService represents an external service
type ExternalService struct {
	Service
	HealthPath     string `json:"health_path"`
	HealthPort     int    `json:"health_port"`
	URLWithoutPort string `json:"url_without_port"`
	Port           int    `json:"port"`
}
