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
