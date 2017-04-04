package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/olekukonko/tablewriter"
	"github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/viper"
)

const (
	viperDBUser     = "db.user"
	viperDBPassword = "db.password"
	viperDBName     = "db.name"
	viperDBHost     = "db.host"
	viperDBPort     = "db.port"
	viperDBSSLMode  = "db.sslmode"
	viperDBMaxConn  = "db.maxconn"
	viperDBTimeout  = "db.timeout"
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
	cfgFile             string
	sqlMigrateDir       string
	sqlMigrateDryRun    bool
	sqlMigrateLimitUp   int
	sqlMigrateLimitDown int
)

func setFlags(cmd *cobra.Command) {
	pflags := cmd.Flags()
	pflags.String("db-user", "cds", "DB User")
	pflags.String("db-password", "", "DB Password")
	pflags.String("db-name", "cds", "DB Name")
	pflags.String("db-host", "localhost", "DB Host")
	pflags.String("db-port", "5432", "DB Port")
	pflags.String("db-sslmode", "require", "DB SSL Mode: require (default), verify-full, or disable")
	pflags.Int("db-maxconn", 20, "DB Max connection")
	pflags.Int("db-timeout", 3000, "Statement timeout value")
	viper.BindPFlag(viperDBUser, pflags.Lookup("db-user"))
	viper.BindPFlag(viperDBPassword, pflags.Lookup("db-password"))
	viper.BindPFlag(viperDBName, pflags.Lookup("db-name"))
	viper.BindPFlag(viperDBHost, pflags.Lookup("db-host"))
	viper.BindPFlag(viperDBPort, pflags.Lookup("db-port"))
	viper.BindPFlag(viperDBSSLMode, pflags.Lookup("db-sslmode"))
	viper.BindPFlag(viperDBMaxConn, pflags.Lookup("db-maxconn"))
	viper.BindPFlag(viperDBTimeout, pflags.Lookup("db-timeout"))
}

func init() {
	setFlags(upgradeCmd)
	setFlags(downgradeCmd)
	setFlags(statusCmd)
	DBCmd.AddCommand(upgradeCmd)
	DBCmd.AddCommand(downgradeCmd)
	DBCmd.AddCommand(statusCmd)

	upgradeCmd.Flags().StringVarP(&sqlMigrateDir, "migrate-dir", "", "./engine/sql", "CDS SQL Migration directory")
	upgradeCmd.Flags().BoolVarP(&sqlMigrateDryRun, "dry-run", "", false, "Dry run upgrade")
	upgradeCmd.Flags().IntVarP(&sqlMigrateLimitUp, "limit", "", 0, "Max number of migrations to apply (0 = unlimited)")

	downgradeCmd.Flags().StringVarP(&sqlMigrateDir, "migrate-dir", "", "./engine/sql", "CDS SQL Migration directory")
	downgradeCmd.Flags().BoolVarP(&sqlMigrateDryRun, "dry-run", "", false, "Dry run downgrade")
	downgradeCmd.Flags().IntVarP(&sqlMigrateLimitDown, "limit", "", 1, "Max number of migrations to apply (0 = unlimited)")

	statusCmd.Flags().StringVarP(&sqlMigrateDir, "migrate-dir", "", "./engine/sql", "CDS SQL Migration directory")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName("api.config") // name of config file (without extension)
	viper.AddConfigPath("$HOME/.cds") // adding home directory as first search path
	viper.AutomaticEnv()              // read in environment variables that match
	viper.SetEnvPrefix("cds")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")) // Replace "." and "-" by "_" for env variable lookup

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

type statusRow struct {
	Id        string
	Migrated  bool
	AppliedAt time.Time
}

func upgradeCmdFunc(cmd *cobra.Command, args []string) {
	initConfig()
	if err := ApplyMigrations(migrate.Up, sqlMigrateDryRun, sqlMigrateLimitUp); err != nil {
		sdk.Exit("Error: %s\n", err)
	}
}

func downgradeCmdFunc(cmd *cobra.Command, args []string) {
	initConfig()
	if err := ApplyMigrations(migrate.Down, sqlMigrateDryRun, sqlMigrateLimitDown); err != nil {
		sdk.Exit("Error: %s\n", err)
	}
}

func statusCmdFunc(cmd *cobra.Command, args []string) {
	initConfig()
	db, err := Init(
		viper.GetString(viperDBUser),
		viper.GetString(viperDBPassword),
		viper.GetString(viperDBName),
		viper.GetString(viperDBHost),
		viper.GetString(viperDBPort),
		viper.GetString(viperDBSSLMode),
		viper.GetInt(viperDBTimeout),
		viper.GetInt(viperDBMaxConn))
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
	db, err := Init(
		viper.GetString(viperDBUser),
		viper.GetString(viperDBPassword),
		viper.GetString(viperDBName),
		viper.GetString(viperDBHost),
		viper.GetString(viperDBPort),
		viper.GetString(viperDBSSLMode),
		viper.GetInt(viperDBTimeout),
		viper.GetInt(viperDBMaxConn))
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
