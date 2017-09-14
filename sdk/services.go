package sdk

import "time"

type Service struct {
	Name          string    `json:"name" db:"name"`
	Type          string    `json:"type" db:"type"`
	HTTPURL       string    `json:"http_url" db:"http_url"`
	LastHeartbeat time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	Hash          string    `json:"hash" db:"hash"`
	Token         string    `json:"token" db:"-"`
}
