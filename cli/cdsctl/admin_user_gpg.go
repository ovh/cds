package main

import (
	"context"
	"github.com/ovh/cds/cli"
	"github.com/spf13/cobra"
)

var adminUsersGPGCmd = cli.Command{
	Name:    "gpg",
	Aliases: []string{"pgp"},
	Short:   "Manage GPG keys",
}

func adminUsersGPG() *cobra.Command {
	return cli.NewCommand(adminUsersGPGCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminUserGpgKeyListCmd, adminUserGpgKeyList, nil),
		cli.NewCommand(adminUserDeleteGPGKeyCmd, adminUserDeleteGPGKey, nil),
	})
}

var adminUserDeleteGPGKeyCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete a user's GPG key",
	Args: []cli.Arg{
		{
			Name: "username",
		},
		{
			Name: "gpg-key-id",
		},
	},
}

func adminUserDeleteGPGKey(v cli.Values) error {
	if err := client.UserGpgKeyDelete(context.Background(), v.GetString("username"), v.GetString("gpg-key-id")); err != nil {
		return err
	}
	return nil
}

var adminUserGpgKeyListCmd = cli.Command{
	Name:  "list",
	Short: "List user's gpg keys",
	Args: []cli.Arg{
		{
			Name: "username",
		},
	},
}

func adminUserGpgKeyList(v cli.Values) (cli.ListResult, error) {
	keys, err := client.UserGpgKeyList(context.Background(), v.GetString("username"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(keys), nil
}
