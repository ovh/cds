package main

import (
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var adminMigrationsCmd = cli.Command{
	Name:    "migration",
	Aliases: []string{"migrations"},
	Short:   "Manage CDS Migrations",
	Long: `Theses commands manage CDS Migration and DO NOT concern database migrations.
	
A CDS Migration is an internal routine. This helps manage a complex data migration with code included
in CDS Engine. It's totally transpartent to CDS Users & Administrators - but these commands can help
CDS Administrators and core CDS Developers to debug something if needed.
	`,
}

func adminMigrations() *cobra.Command {
	return cli.NewCommand(adminMigrationsCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminMigrationsList, adminMigrationsListFunc, nil),
		cli.NewCommand(adminMigrationsCancel, adminMigrationsCancelFunc, nil),
		cli.NewCommand(adminMigrationsReset, adminMigrationsResetFunc, nil),
	})
}

var adminMigrationsList = cli.Command{
	Name:  "list",
	Short: "List all CDS migrations and their states",
}

func adminMigrationsListFunc(_ cli.Values) (cli.ListResult, error) {
	migrations, err := client.AdminCDSMigrationList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(migrations), nil
}

var adminMigrationsCancel = cli.Command{
	Name:  "cancel",
	Short: "Cancel a CDS migration (USE WITH CAUTION)",
	Args: []cli.Arg{
		{Name: "id"},
	},
}

func adminMigrationsCancelFunc(v cli.Values) error {
	id, err := v.GetInt64("id")
	if err != nil {
		return sdk.WrapError(err, "Bad id format")
	}

	if err := client.AdminCDSMigrationCancel(id); err != nil {
		return err
	}
	fmt.Printf("Migration %d is canceled\n", id)
	return nil
}

var adminMigrationsReset = cli.Command{
	Name:  "reset",
	Short: `Reset a CDS migration, so basically it put the migration status to "TO DO" (USE WITH CAUTION)`,
	Args: []cli.Arg{
		{Name: "id"},
	},
}

func adminMigrationsResetFunc(v cli.Values) error {
	id, err := v.GetInt64("id")
	if err != nil {
		return sdk.WrapError(err, "Bad id format")
	}

	if err := client.AdminCDSMigrationReset(id); err != nil {
		return err
	}
	fmt.Printf("Migration %d is reset\n", id)
	return nil
}
