package sdk

import "time"

const (
	// MigrationStatusTodo is the constant to indicate that the migration is "to do"
	MigrationStatusTodo string = "TODO"
	// MigrationStatusInProgress is the constant to indicate that the migration is "in progress"
	MigrationStatusInProgress string = "IN PROGRESS"
	// MigrationStatusDone is the constant to indicate that the migration is "done"
	MigrationStatusDone string = "DONE"
	// MigrationStatusCanceled is the constant to indicate that the migration is "canceled"
	MigrationStatusCanceled string = "CANCELED"
)

// Migration represent a CDS migration
type Migration struct {
	ID        int64     `json:"id" db:"id" cli:"id"`
	Name      string    `json:"name" db:"name" cli:"name"`
	Status    string    `json:"status" db:"status" cli:"status"`
	Release   string    `json:"release" db:"release" cli:"release"`
	Progress  string    `json:"progress" db:"progress" cli:"progress"`
	Error     string    `json:"error" db:"error" cli:"error"`
	Mandatory bool      `json:"mandatory" db:"mandatory" cli:"mandatory"`
	Created   time.Time `json:"created" db:"created" cli:"created"`
	Done      time.Time `json:"done" db:"done" cli:"done"`
}
