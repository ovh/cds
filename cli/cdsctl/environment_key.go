package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	environmentKeyCmd = cli.Command{
		Name:  "keys",
		Short: "Manage CDS environment keys",
	}

	environmentKey = cli.NewCommand(environmentKeyCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(environmentKeyCreateCmd, environmentCreateKeyRun, nil),
			cli.NewListCommand(environmentKeyListCmd, environmentListKeyRun, nil),
			cli.NewCommand(environmentKeyDeleteCmd, environmentDeleteKeyRun, nil),
		})
)

var environmentKeyCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new key on environment. key type can be ssh or pgp",
	Args: []cli.Arg{
		{Name: "key"},
		{Name: "envName"},
		{Name: "keyName"},
		{Name: "keyType"},
	},
}

func environmentCreateKeyRun(v cli.Values) error {
	key := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: v["keyName"],
			Type: v["keyType"],
		},
	}
	if err := client.EnvironmentKeyCreate(v["key"], v["envName"], key); err != nil {
		return err
	}
	return nil
}

var environmentKeyListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS environment keys",
	Args: []cli.Arg{
		{Name: "key"},
		{Name: "envName"},
	},
}

func environmentListKeyRun(v cli.Values) (cli.ListResult, error) {
	keys, err := client.EnvironmentKeysList(v["key"], v["envName"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(keys), nil
}

var environmentKeyDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS environment key",
	Args: []cli.Arg{
		{Name: "key"},
		{Name: "envName"},
		{Name: "keyName"},
	},
}

func environmentDeleteKeyRun(v cli.Values) error {
	return client.EnvironmentKeysDelete(v["key"], v["envName"], v["keyName"])
}
