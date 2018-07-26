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
	LastBeat      time.Time `json:"-"`
	Model         Model     `json:"model"`
	Version       string    `json:"version"`
	Uptodate      bool      `json:"up_to_date"`
	IsSharedInfra bool      `json:"is_shared_infra"`
}
