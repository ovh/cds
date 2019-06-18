package sdk

import "time"

type Result struct {
	ID           int64      `json:"id,omitempty"`
	BuildID      int64      `json:"buildID,omitempty"`
	Status       string     `json:"status,omitempty"`
	Version      int64      `json:"version,omitempty"`
	Reason       string     `json:"reason,omitempty"`
	RemoteTime   time.Time  `json:"remoteTime,omitempty"`
	Duration     string     `json:"duration,omitempty"`
	NewVariables []Variable `json:"new_variables,omitempty"`
}
