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
	Short: "Unlock a pending migration",
	Args: []cli.Arg{
		{Name: "id"},
	},
}

func adminDatabaseUnlockFunc(v cli.Values) error {
	res, err := client.AdminDatabaseMigrationUnlock(v.GetString("id"))
	if err != nil {
		return err
	}
	if len(res) > 0 {
		println(res)
	}
	return nil
}
