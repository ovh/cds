package sdk

import (
	"time"
)

// Result refers to an build result after completion
type Result struct {
	ID         int64     `json:"id" yaml:"-"`
	BuildID    int64     `json:"build_id" yaml:"build"`
	Status     Status    `json:"status"`
	Version    int64     `json:"version"`
	Reason     string    `json:"reason"`
	RemoteTime time.Time `json:"remote_time"`
	Duration   string    `json:"duration"`
}
