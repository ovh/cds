package database

import (
	"fmt"
	"os"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/olekukonko/tablewriter"
	"github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"

	"database/sql"
	"github.com/ovh/cds/sdk"
)

//DBCmd is the root command for database management
var DBCmd = &cobra.Command{
	Use:   "database",
	Short: "Manage CDS database",
	Long:  "Manage CDS database",
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade schema",
	Long:  "Migrates the database to the most recent version available.",
	Run:   upgradeCmdFunc,
}

var downgradeCmd = &cobra.Command{
	Use:   "downgrade",
	Short: "Downgrade schema",
	Long:  "Undo a database migration.",
	Run:   downgradeCmdFunc,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current migration status",
	Long:  "",
	Run:   statusCmdFunc,
}

var (
	sqlMigrateDir    string
	sqlMigrateDryRun bool
	sqlMigrateLimit  int
)

func init() {
	DBCmd.AddCommand(upgradeCmd)
	DBCmd.AddCommand(downgradeCmd)
	DBCmd.AddCommand(statusCmd)

	upgradeCmd.Flags().StringVarP(&sqlMigrateDir, "migrate-dir", "", "./engine/sql", "CDS SQL Migration directory")
	upgradeCmd.Flags().BoolVarP(&sqlMigrateDryRun, "dry-run", "", false, "Dry run upgrade")
	upgradeCmd.Flags().IntVarP(&sqlMigrateLimit, "limit", "", 0, "Max number of migrations to apply (0 = unlimited)")

	downgradeCmd.Flags().StringVarP(&sqlMigrateDir, "migrate-dir", "", "./engine/sql", "CDS SQL Migration directory")
	downgradeCmd.Flags().BoolVarP(&sqlMigrateDryRun, "dry-run", "", false, "Dry run downgrade")
	downgradeCmd.Flags().IntVarP(&sqlMigrateLimit, "limit", "", 1, "Max number of migrations to apply (0 = unlimited)")

	statusCmd.Flags().StringVarP(&sqlMigrateDir, "migrate-dir", "", "./engine/sql", "CDS SQL Migration directory")
}

type statusRow struct {
	Id        string
	Migrated  bool
	AppliedAt time.Time
}

func upgradeCmdFunc(cmd *cobra.Command, args []string) {
	if err := ApplyMigrations(migrate.Up, sqlMigrateDryRun, sqlMigrateLimit); err != nil {
		sdk.Exit("Error: %s\n", err)
	}
}

func downgradeCmdFunc(cmd *cobra.Command, args []string) {
	if err := ApplyMigrations(migrate.Down, sqlMigrateDryRun, sqlMigrateLimit); err != nil {
		sdk.Exit("Error: %s\n", err)
	}
}

func statusCmdFunc(cmd *cobra.Command, args []string) {
	db, err := Init()
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	source := migrate.FileMigrationSource{
		Dir: sqlMigrateDir,
	}

	migrations, err := source.FindMigrations()
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	records, err := migrate.GetMigrationRecords(db, "postgres")
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Migration", "Applied"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetColWidth(60)

	rows := make(map[string]*statusRow)

	for _, m := range migrations {
		rows[m.Id] = &statusRow{
			Id:       m.Id,
			Migrated: false,
		}
	}

	for _, r := range records {
		rows[r.Id].Migrated = true
		rows[r.Id].AppliedAt = r.AppliedAt
	}

	for _, m := range migrations {
		if rows[m.Id].Migrated {
			table.Append([]string{
				m.Id,
				rows[m.Id].AppliedAt.String(),
			})
		} else {
			table.Append([]string{
				m.Id,
				"no",
			})
		}
	}

	table.Render()

}

//ApplyMigrations applies migration (or not depending on dryrun flag)
func ApplyMigrations(dir migrate.MigrationDirection, dryrun bool, limit int) error {
	db, err := Init()
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	source := migrate.FileMigrationSource{
		Dir: sqlMigrateDir,
	}

	if dryrun {
		migrations, _, err := migrate.PlanMigration(db, "postgres", source, dir, limit)
		if err != nil {
			return fmt.Errorf("Cannot plan migration: %s", err)
		}

		for _, m := range migrations {
			printMigration(m, dir)
		}
		return nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}
	hostname = fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())
	if err := lockMigrate(db, hostname); err != nil {
		sdk.Exit("Unable to lock database: %s\n", err)
	}

	defer unlockMigrate(db, hostname)

	n, err := migrate.ExecMax(db, "postgres", source, dir, limit)
	if err != nil {
		return fmt.Errorf("Migration failed: %s", err)
	}

	if n == 1 {
		fmt.Println("Applied 1 migration")
	} else {
		fmt.Printf("Applied %d migrations\n", n)
	}

	return nil
}

func printMigration(m *migrate.PlannedMigration, dir migrate.MigrationDirection) {
	if dir == migrate.Up {
		fmt.Printf("==> Would apply migration %s (up)\n", m.Id)
		for _, q := range m.Up {
			fmt.Println(q)
		}
	} else if dir == migrate.Down {
		fmt.Printf("==> Would apply migration %s (down)\n", m.Id)
		for _, q := range m.Down {
			fmt.Println(q)
		}
	} else {
		panic("Not reached")
	}
}

//MigrationLock is used to lock the migration (managed by gorp)
type MigrationLock struct {
	Id       string     `db:"id"`
	Locked   *time.Time `db:"locked"`
	Unlocked *time.Time `db:"unlocked"`
}

func lockMigrate(db *sql.DB, id string) error {
	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	dbmap.AddTableWithName(MigrationLock{}, "gorp_migrations_lock").SetKeys(false, "Id")
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
		return fmt.Errorf("Migration is locked by %s since %v", pendingMigration[0].Id, pendingMigration[0].Locked)
	}

	t := time.Now()
	m := MigrationLock{
		Id:     id,
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
	dbmap.AddTableWithName(MigrationLock{}, "gorp_migrations_lock").SetKeys(false, "Id")

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
