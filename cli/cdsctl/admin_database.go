package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var adminDatabaseCmd = cli.Command{
	Name:  "database",
	Short: "Manage CDS Database",
}

func adminDatabase() *cobra.Command {
	return cli.NewCommand(adminDatabaseCmd, nil, []*cobra.Command{
		cli.NewCommand(adminDatabaseUnlockCmd, adminDatabaseUnlockFunc, nil),
		cli.NewCommand(adminDatabaseDeleteMigrationCmd, adminDatabaseDeleteFunc, nil),
		cli.NewListCommand(adminDatabaseMigrationsList, adminDatabaseMigrationsListFunc, nil),
	})
}

var adminDatabaseUnlockCmd = cli.Command{
	Name:  "unlock",
	Short: "Unlock a pending migration (Use with caution)",
	Args: []cli.Arg{
		{Name: "id"},
	},
}

func adminDatabaseUnlockFunc(v cli.Values) error {
	return client.AdminDatabaseMigrationUnlock(v.GetString("id"))
}

var adminDatabaseDeleteMigrationCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a database migration from table gorp_migration (use with caution)",
	Args: []cli.Arg{
		{Name: "id"},
	},
}

func adminDatabaseDeleteFunc(v cli.Values) error {
	return client.AdminDatabaseMigrationDelete(v.GetString("id"))
}

var adminDatabaseMigrationsList = cli.Command{
	Name:  "list",
	Short: "List all CDS DB migrations",
}

func adminDatabaseMigrationsListFunc(_ cli.Values) (cli.ListResult, error) {
	migrations, err := client.AdminDatabaseMigrationsList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(migrations), nil
}
