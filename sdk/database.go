package sdk

import (
	"time"
)

// DatabaseMigrationStatus represents on migration script status
type DatabaseMigrationStatus struct {
	ID        string     `json:"id" db:"id" cli:"id,key"`
	Migrated  bool       `json:"migrated" db:"-" cli:"migrated"`
	AppliedAt *time.Time `json:"applied_at" db:"applied_at" cli:"applied_at"`
}
