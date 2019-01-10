package database

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mholt/archiver"
	"github.com/olekukonko/tablewriter"
	"github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/api/database/dbmigrate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
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
	Long:  `Migrates the database to the most recent version available.`,
	Example: `engine database upgrade --db-password=your-password --db-sslmode=disable --db-name=cds --migrate-dir=./sql

# If the directory --migrate-dir is not up to date with the current version, this
# directory will be automatically updated with the release from https://github.com/ovh/cds/releases
	`,
	Run: upgradeCmdFunc,
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
	pflags.StringVarP(&connFactory.dbRole, "db-role", "", "", "DB Role")
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
	source := migrate.FileMigrationSource{
		Dir: sqlMigrateDir,
	}

	migrations, err := source.FindMigrations()
	if err != nil {
		sdk.Exit("Error: %v\n", err)
	}

	if sdk.VERSION != "snapshot" {
		nbMigrateOnThisVersion, err := strconv.Atoi(sdk.DBMIGRATE)
		if err == nil { // no err -> it's a real release
			fmt.Printf("There are %d migrate files locally. Current engine needs %d files\n", len(migrations), nbMigrateOnThisVersion)
			if nbMigrateOnThisVersion > len(migrations) {
				fmt.Printf("This version %s should contains %d migrate files.\n", sdk.VERSION, nbMigrateOnThisVersion)
				if err := downloadSQLTarGz(sdk.VERSION, "sql.tar.gz", sqlMigrateDir); err != nil {
					sdk.Exit("Error on download sql.tar.gz: %v\n", err)
				}
				migrations, err := source.FindMigrations()
				if err != nil {
					sdk.Exit("Error: %v\n", err)
				}
				fmt.Printf("sql.tar.gz downloaded, there are now %d migrate files locally\n", len(migrations))
			}
		} else {
			sdk.Exit("Invalid compilation flag DBMIGRATE")
		}
	}

	if err := ApplyMigrations(migrate.Up, sqlMigrateDryRun, sqlMigrateLimitUp); err != nil {
		sdk.Exit("Error: %v\n", err)
	}
}

// downloadSQLTarGz downloads sql.tar.gz from github release corresponding to the current engine version
// check status 200 on download
// then write sql.tar.gz file in tmpdir
// then unzip sql.tar.gz file
// then move all sql file to sqlMigrateDir directory
func downloadSQLTarGz(currentVersion string, artifactName string, migrateDir string) error {
	if !strings.Contains(currentVersion, "+") {
		return fmt.Errorf("invalid current version: %s, ersion must contains a '+'", currentVersion)
	}
	if _, err := os.Stat(migrateDir); os.IsNotExist(err) {
		return fmt.Errorf("--migrate-dir does not exist: %s", migrateDir)
	}
	currentTag := currentVersion[:strings.LastIndex(currentVersion, "+")]
	urlTarGZ := fmt.Sprintf("https://github.com/ovh/cds/releases/download/%s/%s", currentTag, artifactName)
	fmt.Printf("Getting %s from github on %s...\n", artifactName, urlTarGZ)

	httpClient := cdsclient.NewHTTPClient(60*time.Second, false)
	resp, err := httpClient.Get(urlTarGZ)
	if err != nil {
		return fmt.Errorf("error while getting %s from Github: %v", artifactName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(resp.Body); err == nil {
			fmt.Printf("body returned from github: \n%s\n", buf.String())
		}
		return fmt.Errorf("error http code: %d, url called: %s", resp.StatusCode, urlTarGZ)
	}

	if err := sdk.CheckContentTypeBinary(resp); err != nil {
		return fmt.Errorf("invalid content type: %s", err.Error())
	}

	tmpfile, err := ioutil.TempFile(os.TempDir(), artifactName)
	if err != nil {
		sdk.Exit(err.Error())
	}
	defer tmpfile.Close()

	if _, err = io.Copy(tmpfile, resp.Body); err != nil {
		return fmt.Errorf("error on creating file: %v", err.Error())
	}

	dest := tmpfile.Name() + "extract"
	if err := archiver.TarGz.Open(tmpfile.Name(), dest); err != nil {
		return fmt.Errorf("Unarchive %s failed: %v", artifactName, err)
	}
	// the directory dest/sql/ -> contains all sql files
	fmt.Printf("Unarchive to %s\n", dest)

	dirFiles := dest + "/sql"
	files, err := ioutil.ReadDir(dirFiles)
	if err != nil {
		return fmt.Errorf("error on readDir %s", dirFiles)
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".sql") {
			src := dirFiles + "/" + f.Name()
			dest := migrateDir + "/" + filepath.Base(f.Name())
			if err := os.Rename(src, dest); err != nil {
				return fmt.Errorf("error on move %s to %s err:%v", src, dest, err)
			}
		}
	}
	return nil
}

func downgradeCmdFunc(cmd *cobra.Command, args []string) {
	if err := ApplyMigrations(migrate.Down, sqlMigrateDryRun, sqlMigrateLimitDown); err != nil {
		sdk.Exit("Error: %v\n", err)
	}
}

func statusCmdFunc(cmd *cobra.Command, args []string) {
	var err error
	connFactory, err = Init(connFactory.dbUser, connFactory.dbRole, connFactory.dbPassword, connFactory.dbName, connFactory.dbHost, connFactory.dbPort, connFactory.dbSSLMode, connFactory.dbConnectTimeout, connFactory.dbTimeout, connFactory.dbMaxConn)
	if err != nil {
		sdk.Exit("Error: %v\n", err)
	}

	source := migrate.FileMigrationSource{
		Dir: sqlMigrateDir,
	}

	migrations, err := source.FindMigrations()
	if err != nil {
		sdk.Exit("Error: %v\n", err)
	}

	records, err := migrate.GetMigrationRecords(connFactory.DB(), "postgres")
	if err != nil {
		sdk.Exit("Error: %v\n", err)
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
	connFactory, err = Init(connFactory.dbUser, connFactory.dbRole, connFactory.dbPassword, connFactory.dbName, connFactory.dbHost, connFactory.dbPort, connFactory.dbSSLMode, connFactory.dbConnectTimeout, connFactory.dbTimeout, connFactory.dbMaxConn)
	if err != nil {
		sdk.Exit("Error: %v\n", err)
	}

	migrations, err := dbmigrate.Do(connFactory.DB, sqlMigrateDir, dir, dryrun, limit)
	if err != nil {
		sdk.Exit("Error: %v\n", err)
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
