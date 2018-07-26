package sdk

import (
	"time"
)

// Broadcast represents a message (communication CDS admins -> users)
type Broadcast struct {
	ID         int64     `json:"id" db:"id" cli:"num,key"`
	Title      string    `json:"title" db:"title" cli:"title"`
	Content    string    `json:"content" db:"content" cli:"content"`
	Level      string    `json:"level" db:"level" cli:"level"`
	ProjectKey string    `json:"project_key,omitempty" db:"-" cli:"project"`
	Created    time.Time `json:"created" db:"created" cli:"created"`
	Updated    time.Time `json:"updated" db:"updated" cli:"-"`
	ProjectID  *int64    `json:"project_id,omitempty" db:"project_id" cli:"-"`
	Archived   bool      `json:"archived" db:"archived" cli:"archived"`
	Read       bool      `json:"read" db:"-" cli:"read"`
}
