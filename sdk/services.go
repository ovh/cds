package sdk

import "time"

// Service is a µService registered on CDS API
type Service struct {
	Name          string    `json:"name" db:"name" cli:"name,key"`
	Type          string    `json:"type" db:"type" cli:"type"`
	HTTPURL       string    `json:"http_url" db:"http_url" cli:"url"`
	LastHeartbeat time.Time `json:"last_heartbeat" db:"last_heartbeat" cli:"heartbeat"`
	Hash          string    `json:"hash" db:"hash"`
	Token         string    `json:"token" db:"-"`
}
