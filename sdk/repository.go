package sdk

import (
	"time"
)

type ProjectRepository struct {
	ID           string    `json:"-" db:"id"`
	Name         string    `json:"name" db:"name" cli:"name,key"`
	Created      time.Time `json:"created" db:"created"`
	CreatedBy    string    `json:"created_by" db:"created_by"`
	VCSProjectID string    `json:"-" db:"vcs_project_id"`
}
