package dbmigrate

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/ovh/cds/sdk"

	gorp "github.com/go-gorp/gorp"
	migrate "github.com/rubenv/sql-migrate"
)

//Do applies migration
func Do(DBFunc func() *sql.DB, dialect gorp.Dialect, sqlMigrateDir string, dir migrate.MigrationDirection, dryrun bool, limit int) ([]*migrate.PlannedMigration, error) {
	source := migrate.FileMigrationSource{
		Dir: sqlMigrateDir,
	}

	if dryrun {
		migrations, _, err := migrate.PlanMigration(DBFunc(), "postgres", source, dir, limit)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot plan migration")
		}

		return migrations, nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	hostname = fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())
	if err := lockMigrate(DBFunc(), hostname, dialect); err != nil {
		return nil, sdk.WithStack(err)
	}

	_, errExec := migrate.ExecMax(DBFunc(), "postgres", source, dir, limit)

	if err := UnlockMigrate(DBFunc(), hostname, dialect); err != nil {
		return nil, sdk.WrapError(err, "cannot unlock migration")
	}

	return nil, errExec
}

// MigrationLock is used to lock the migration (managed by gorp)
type MigrationLock struct {
	ID       string     `db:"id"`
	Locked   *time.Time `db:"locked"`
	Unlocked *time.Time `db:"unlocked"`
}

// DatabaseMigration represents an entry in table gorp_migrations
type DatabaseMigration struct {
	ID        string     `db:"id"`
	AppliedAt *time.Time `db:"applied_at"`
}

func lockMigrate(db *sql.DB, id string, dialect gorp.Dialect) error {
	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: dialect}
	dbmap.AddTableWithName(MigrationLock{}, "gorp_migrations_lock").SetKeys(false, "ID")
	// create table if not exist
	if err := dbmap.CreateTablesIfNotExists(); err != nil {
		return sdk.WithStack(err)
	}

	tx, err := dbmap.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	var pendingMigration []MigrationLock
	var query string
	switch dialect.(type) {
	case gorp.PostgresDialect:
		query = "SELECT * FROM gorp_migrations_lock WHERE unlocked IS NULL FOR UPDATE OF gorp_migrations_lock NOWAIT"
	default:
		query = "SELECT * FROM gorp_migrations_lock WHERE unlocked IS NULL"
	}

	if _, err := tx.Select(&pendingMigration, query); err != nil {
		return sdk.WithStack(err)
	}

	if len(pendingMigration) > 0 {
		return sdk.WithStack(fmt.Errorf("Migration is locked by %s since %v", pendingMigration[0].ID, pendingMigration[0].Locked))
	}

	t := time.Now()
	m := MigrationLock{
		ID:     id,
		Locked: &t,
	}

	if err := tx.Insert(&m); err != nil {
		return sdk.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// DeleteMigrate delete an ID from table gorp_migrations
func DeleteMigrate(db *sql.DB, id string) error {
	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	dbmap.AddTableWithName(sdk.DatabaseMigrationStatus{}, "gorp_migrations").SetKeys(false, "ID")

	tx, err := dbmap.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	m := sdk.DatabaseMigrationStatus{}
	if err := tx.SelectOne(&m, "SELECT * FROM gorp_migrations WHERE id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return sdk.WithStack(sdk.ErrNoDBMigrationID)
		}
		return sdk.WithStack(err)
	}

	if _, err := tx.Delete(&m); err != nil {
		return sdk.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// List list current entries from gorp_migration only
func List(db *sql.DB) ([]sdk.DatabaseMigrationStatus, error) {
	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	dbmap.AddTableWithName(sdk.DatabaseMigrationStatus{}, "gorp_migrations").SetKeys(false, "ID")

	tx, err := dbmap.Begin()
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	var migrationsApplied []sdk.DatabaseMigrationStatus
	if _, err := tx.Select(&migrationsApplied, "SELECT * FROM gorp_migrations"); err != nil {
		return nil, sdk.WithStack(err)
	}

	for i := range migrationsApplied {
		// line is in table, it's applied
		migrationsApplied[i].Migrated = true
	}
	return migrationsApplied, nil
}

// UnlockMigrate unlocks an ID from table gorp_migrations_lock
func UnlockMigrate(db *sql.DB, id string, dialect gorp.Dialect) error {
	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: dialect}
	dbmap.AddTableWithName(MigrationLock{}, "gorp_migrations_lock").SetKeys(false, "ID")

	tx, err := dbmap.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	var pendingMigration []MigrationLock
	var query string
	switch dialect.(type) {
	case gorp.PostgresDialect:
		query = "SELECT * FROM gorp_migrations_lock WHERE unlocked IS NULL FOR UPDATE OF gorp_migrations_lock NOWAIT"
	default:
		query = "SELECT * FROM gorp_migrations_lock WHERE unlocked IS NULL"
	}

	if _, err := tx.Select(&pendingMigration, query); err != nil {
		return sdk.WithStack(err)
	}

	if len(pendingMigration) == 0 {
		return fmt.Errorf("There is no migration to unlock")
	}

	m := MigrationLock{}
	if err := tx.SelectOne(&m, "SELECT * FROM gorp_migrations_lock WHERE id = $1", id); err != nil {
		return sdk.WithStack(err)
	}

	t := time.Now()
	m.Unlocked = &t

	if _, err := tx.Update(&m); err != nil {
		return sdk.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// Get the status of all migration scripts
func Get(DBFunc func() *sql.DB, dir string, dialect gorp.Dialect) ([]sdk.DatabaseMigrationStatus, error) {
	source := migrate.FileMigrationSource{
		Dir: dir,
	}

	migrations, err := source.FindMigrations()
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	var dialectString = "postgres"
	switch dialect.(type) {
	case gorp.SqliteDialect:
		dialectString = "sqlite3"
	}

	records, err := migrate.GetMigrationRecords(DBFunc(), dialectString)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	rows := make(map[string]sdk.DatabaseMigrationStatus)
	for _, m := range migrations {
		rows[m.Id] = sdk.DatabaseMigrationStatus{
			ID:       m.Id,
			Migrated: false,
		}
	}

	for _, r := range records {
		if _, ok := rows[r.Id]; !ok {
			return nil, fmt.Errorf("record '%s' not in migration list, manual migration needed", r.Id)
		}
		s := rows[r.Id]
		s.Migrated = true
		s.AppliedAt = &r.AppliedAt
		rows[r.Id] = s
	}

	res := make([]sdk.DatabaseMigrationStatus, len(rows))
	var i int
	for _, r := range rows {
		res[i] = r
		i++
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ID < res[j].ID
	})

	return res, nil
}
