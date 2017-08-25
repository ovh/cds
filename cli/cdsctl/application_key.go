package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	applicationKeyCmd = cli.Command{
		Name:  "keys",
		Short: "Manage CDS application keys",
	}

	applicationKey = cli.NewCommand(applicationKeyCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(applicationKeyCreateCmd, applicationCreateKeyRun, nil),
			cli.NewListCommand(applicationKeyListCmd, applicationListKeyRun, nil),
			cli.NewCommand(applicationKeyDeleteCmd, applicationDeleteKeyRun, nil),
		})
)

var applicationKeyCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new key on application. key type can be ssh or pgp",
	Args: []cli.Arg{
		{Name: "key"},
		{Name: "appName"},
		{Name: "keyName"},
		{Name: "keyType"},
	},
}

func applicationCreateKeyRun(v cli.Values) error {
	key := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: v["keyName"],
			Type: v["keyType"],
		},
	}
	if err := client.ApplicationKeyCreate(v["key"], v["appName"], key); err != nil {
		return err
	}
	return nil
}

var applicationKeyListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS application keys",
	Args: []cli.Arg{
		{Name: "key"},
		{Name: "appName"},
	},
}

func applicationListKeyRun(v cli.Values) (cli.ListResult, error) {
	keys, err := client.ApplicationKeysList(v["key"], v["appName"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(keys), nil
}

var applicationKeyDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS an application key",
	Args: []cli.Arg{
		{Name: "key"},
		{Name: "appName"},
		{Name: "keyName"},
	},
}

func applicationDeleteKeyRun(v cli.Values) error {
	return client.ApplicationKeysDelete(v["key"], v["appName"], v["keyName"])
}
