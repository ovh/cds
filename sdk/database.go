package sdk

import (
	"time"
)

// DatabaseMigrationStatus represents on migration script status
type DatabaseMigrationStatus struct {
	ID        string     `json:"id" db:"id" cli:"id,key"`
	Migrated  bool       `json:"migrated" db:"-" cli:"migrated"`
	AppliedAt *time.Time `json:"applied_at" db:"applied_at" cli:"applied_at"`
	// aggregates
	Database string `json:"database" db:"-" cli:"database"`
}

type CanonicalFormUsage struct {
	Signer string `json:"signer" db:"signer"`
	Number int64  `json:"number" db:"number"`
	Latest bool   `json:"latest"`
}

type CanonicalFormUsageResume map[string][]CanonicalFormUsage
