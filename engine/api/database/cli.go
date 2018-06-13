package database

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/api/database/dbmigrate"
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
	Run:   statusCmdFunc,
}

var (
	sqlMigrateDir       string
	sqlMigrateDryRun    bool
	sqlMigrateLimitUp   int
	sqlMigrateLimitDown int
	connFactory         = &DBConnectionFactory{}
)

func setFlags(cmd *cobra.Command) {
	pflags := cmd.Flags()
	pflags.StringVarP(&connFactory.dbUser, "db-user", "", "cds", "DB User")
	pflags.StringVarP(&connFactory.dbPassword, "db-password", "", "", "DB Password")
	pflags.StringVarP(&connFactory.dbName, "db-name", "", "cds", "DB Name")
	pflags.StringVarP(&connFactory.dbHost, "db-host", "", "localhost", "DB Host")
	pflags.IntVarP(&connFactory.dbPort, "db-port", "", 5432, "DB Port")
	pflags.StringVarP(&sqlMigrateDir, "migrate-dir", "", "./engine/sql", "CDS SQL Migration directory")
	pflags.StringVarP(&connFactory.dbSSLMode, "db-sslmode", "", "require", "DB SSL Mode: require (default), verify-full, or disable")
	pflags.IntVarP(&connFactory.dbMaxConn, "db-maxconn", "", 20, "DB Max connection")
	pflags.IntVarP(&connFactory.dbTimeout, "db-timeout", "", 3000, "Statement timeout value in milliseconds")
	pflags.IntVarP(&connFactory.dbConnectTimeout, "db-connect-timeout", "", 10, "Maximum wait for connection, in seconds")
}

func init() {
	setFlags(upgradeCmd)
	setFlags(downgradeCmd)
	setFlags(statusCmd)
	DBCmd.AddCommand(upgradeCmd)
	DBCmd.AddCommand(downgradeCmd)
	DBCmd.AddCommand(statusCmd)

	upgradeCmd.Flags().BoolVarP(&sqlMigrateDryRun, "dry-run", "", false, "Dry run upgrade")
	upgradeCmd.Flags().IntVarP(&sqlMigrateLimitUp, "limit", "", 0, "Max number of migrations to apply (0 = unlimited)")

	downgradeCmd.Flags().BoolVarP(&sqlMigrateDryRun, "dry-run", "", false, "Dry run downgrade")
	downgradeCmd.Flags().IntVarP(&sqlMigrateLimitDown, "limit", "", 1, "Max number of migrations to apply (0 = unlimited)")
}

type statusRow struct {
	ID        string
	Migrated  bool
	AppliedAt time.Time
}

func upgradeCmdFunc(cmd *cobra.Command, args []string) {
	if err := ApplyMigrations(migrate.Up, sqlMigrateDryRun, sqlMigrateLimitUp); err != nil {
		sdk.Exit("Error: %s\n", err)
	}
}

func downgradeCmdFunc(cmd *cobra.Command, args []string) {
	if err := ApplyMigrations(migrate.Down, sqlMigrateDryRun, sqlMigrateLimitDown); err != nil {
		sdk.Exit("Error: %s\n", err)
	}
}

func statusCmdFunc(cmd *cobra.Command, args []string) {
	var err error
	connFactory, err = Init(connFactory.dbUser, connFactory.dbPassword, connFactory.dbName, connFactory.dbHost, connFactory.dbPort, connFactory.dbSSLMode, connFactory.dbConnectTimeout, connFactory.dbTimeout, connFactory.dbMaxConn)
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

	records, err := migrate.GetMigrationRecords(connFactory.DB(), "postgres")
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
			ID:       m.Id,
			Migrated: false,
		}
	}

	for _, r := range records {
		if _, ok := rows[r.Id]; !ok {
			fmt.Printf("Record '%s' not in migration list, manual migration needed\n", r.Id)
			continue
		}
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
	var err error
	connFactory, err = Init(connFactory.dbUser, connFactory.dbPassword, connFactory.dbName, connFactory.dbHost, connFactory.dbPort, connFactory.dbSSLMode, connFactory.dbConnectTimeout, connFactory.dbTimeout, connFactory.dbMaxConn)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	migrations, err := dbmigrate.Do(connFactory.DB, sqlMigrateDir, dir, dryrun, limit)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	if dryrun {
		for _, m := range migrations {
			printMigration(m, dir)
		}
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
