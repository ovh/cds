package sdk

import (
	"time"
)

// Broadcast represents a message (communication CDS admins -> users)
type Broadcast struct {
	ID       int64     `json:"id" db:"id" cli:"num"`
	Title    string    `json:"title" db:"title" cli:"title"`
	Message  string    `json:"message" db:"message" cli:"message"`
	Level    string    `json:"level" db:"level" cli:"level"`
	Created  time.Time `json:"created" db:"created" cli:"-"`
	Updated  time.Time `json:"updated" db:"updated" cli:"-"`
	Archived bool      `json:"archived" db:"archived" cli:"archived"`
}
