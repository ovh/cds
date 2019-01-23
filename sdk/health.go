package sdk

import "time"

// MonDBMigrate is used by /mon/db/migrate
type MonDBMigrate struct {
	ID        string    `db:"id" cli:"id"`
	AppliedAt time.Time `db:"applied_at" cli:"applied_at"`
}
