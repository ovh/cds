package sdk

import (
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
	MonitoringStatus MonitoringStatus `json:"monitoring_status" db:"-" cli:"-"`
	Config           interface{}      `json:"config" db:"-" cli:"-"`
	IsSharedInfra    bool             `json:"is_shared_infra" db:"-"`
	Version          string           `json:"version" db:"-"`
	Uptodate         bool             `json:"up_to_date" db:"-"`
}

// ExternalService represents an external service
type ExternalService struct {
	Service
	HealthURL  string `json:"health_url"`
	HealthPort string `json:"health_port"`
	HealthPath string `json:"health_path"`
	Port       string `json:"port"`
	URL        string `json:"url"`
	Path       string `json:"path"`
}
