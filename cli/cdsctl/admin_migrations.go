package main

import (
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var adminMigrationsCmd = cli.Command{
	Name:  "migrations",
	Short: "Manage CDS Migrations",
}

func adminMigrations() *cobra.Command {
	return cli.NewCommand(adminMigrationsCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminMigrationsList, adminMigrationsListFunc, nil),
		cli.NewCommand(adminMigrationsCancel, adminMigrationsCancelFunc, nil),
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
