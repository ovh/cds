package sdk

import (
	"time"
)

// Hatchery registration model
type Hatchery struct {
	ID            int64     `json:"id"`
	UID           string    `json:"uid"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	GroupID       int64     `json:"group_id"`
	IsSharedInfra bool      `json:"is_shared_infra"`
	LastBeat      time.Time `json:"-"`
	Model         Model     `json:"model"`
	Uptodate      bool      `json:"up_to_date"`
	Version       string    `json:"version"`
}
