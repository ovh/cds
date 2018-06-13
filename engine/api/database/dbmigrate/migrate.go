package dbmigrate

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"time"

	migrate "github.com/rubenv/sql-migrate"
	gorp "gopkg.in/gorp.v1"
)

//Do applies migration
func Do(DBFunc func() *sql.DB, sqlMigrateDir string, dir migrate.MigrationDirection, dryrun bool, limit int) ([]*migrate.PlannedMigration, error) {
	source := migrate.FileMigrationSource{
		Dir: sqlMigrateDir,
	}

	if dryrun {
		migrations, _, err := migrate.PlanMigration(DBFunc(), "postgres", source, dir, limit)
		if err != nil {
			return nil, fmt.Errorf("Cannot plan migration: %s", err)
		}

		return migrations, nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	hostname = fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())
	if err := lockMigrate(DBFunc(), hostname); err != nil {
		return nil, err
	}

	defer unlockMigrate(DBFunc(), hostname)

	_, err = migrate.ExecMax(DBFunc(), "postgres", source, dir, limit)
	return nil, err
}

// MigrationLock is used to lock the migration (managed by gorp)
type MigrationLock struct {
	ID       string     `db:"id"`
	Locked   *time.Time `db:"locked"`
	Unlocked *time.Time `db:"unlocked"`
}

// MigrationStatus represents on migration script status
type MigrationStatus struct {
	ID        string
	Migrated  bool
	AppliedAt time.Time
}

func lockMigrate(db *sql.DB, id string) error {
	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	dbmap.AddTableWithName(MigrationLock{}, "gorp_migrations_lock").SetKeys(false, "ID")
	// create table if not exist
	if err := dbmap.CreateTablesIfNotExists(); err != nil {
		return err
	}

	tx, err := dbmap.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var pendingMigration []MigrationLock
	if _, err := tx.Select(&pendingMigration, "SELECT * FROM gorp_migrations_lock WHERE unlocked IS NULL FOR UPDATE OF gorp_migrations_lock NOWAIT"); err != nil {
		return err
	}

	if len(pendingMigration) > 0 {
		return fmt.Errorf("Migration is locked by %s since %v", pendingMigration[0].ID, pendingMigration[0].Locked)
	}

	t := time.Now()
	m := MigrationLock{
		ID:     id,
		Locked: &t,
	}

	if err := tx.Insert(&m); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func unlockMigrate(db *sql.DB, id string) error {
	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	dbmap.AddTableWithName(MigrationLock{}, "gorp_migrations_lock").SetKeys(false, "ID")

	tx, err := dbmap.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var pendingMigration []MigrationLock
	if _, err := tx.Select(&pendingMigration, "SELECT * FROM gorp_migrations_lock WHERE unlocked IS NULL FOR UPDATE OF gorp_migrations_lock NOWAIT"); err != nil {
		return err
	}

	if len(pendingMigration) == 0 {
		return fmt.Errorf("There is no migration to unlock")
	}

	m := MigrationLock{}
	if err := tx.SelectOne(&m, "SELECT * FROM gorp_migrations_lock WHERE id = $1", id); err != nil {
		return err
	}

	t := time.Now()
	m.Unlocked = &t

	if _, err := tx.Update(&m); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// Get the status of all migration scripts
func Get(DBFunc func() *sql.DB, dir string) ([]MigrationStatus, error) {
	source := migrate.FileMigrationSource{
		Dir: dir,
	}

	migrations, err := source.FindMigrations()
	if err != nil {
		return nil, err
	}

	records, err := migrate.GetMigrationRecords(DBFunc(), "postgres")
	if err != nil {
		return nil, err
	}

	rows := make(map[string]MigrationStatus)
	for _, m := range migrations {
		rows[m.Id] = MigrationStatus{
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
		s.AppliedAt = r.AppliedAt
		rows[r.Id] = s
	}

	res := make([]MigrationStatus, len(rows))
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
