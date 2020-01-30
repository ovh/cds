package sdk

import (
	"context"
	"time"
)

const (
	// MigrationStatusTodo is the constant to indicate that the migration is "to do"
	MigrationStatusTodo string = "TODO"
	// MigrationStatusInProgress is the constant to indicate that the migration is "in progress"
	MigrationStatusInProgress string = "IN PROGRESS"
	// MigrationStatusDone is the constant to indicate that the migration is "done"
	MigrationStatusDone string = "DONE"
	// MigrationStatusCanceled is the constant to indicate that the migration is "canceled"
	MigrationStatusCanceled string = "CANCELED"
	// MigrationStatusNotExecuted is the constant to indicate that the migration is "not executed"
	MigrationStatusNotExecuted string = "NOT EXECUTED"
)

// Migration represent a CDS migration
type Migration struct {
	ID        int64     `json:"id" db:"id" cli:"id"`
	Name      string    `json:"name" db:"name" cli:"name"`
	Status    string    `json:"status" db:"status" cli:"status"`
	Progress  string    `json:"progress" db:"progress" cli:"progress"`
	Error     string    `json:"error" db:"error" cli:"error"`
	Automatic bool      `json:"automatic" db:"mandatory" cli:"automatic"`
	Created   time.Time `json:"created" db:"created" cli:"created"`
	Done      time.Time `json:"done" db:"done" cli:"done"`
	Release   string    `json:"release" db:"release" cli:"release"`
	Major     uint64    `json:"major" db:"major" cli:"major"`
	Minor     uint64    `json:"minor" db:"minor" cli:"minor"`
	Patch     uint64    `json:"patch" db:"patch" cli:"patch"`
	Blocker   bool      `json:"-" db:"-" cli:"-"`

	ExecFunc func(ctx context.Context) error `json:"-" db:"-" cli:"-" yaml:"-"`
}
