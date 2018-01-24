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
			cli.NewCommand(environmentKeyCreateCmd, environmentCreateKeyRun, nil, withAllCommandModifiers()...),
			cli.NewListCommand(environmentKeyListCmd, environmentListKeyRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(environmentKeyDeleteCmd, environmentDeleteKeyRun, nil, withAllCommandModifiers()...),
		})
)

var environmentKeyCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new key on environment. key-type can be ssh or pgp",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
		{Name: "key-name"},
		{Name: "key-type"},
	},
}

func environmentCreateKeyRun(v cli.Values) error {
	key := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: v["key-name"],
			Type: v["key-type"],
		},
	}
	return client.EnvironmentKeyCreate(v[_ProjectKey], v["env-name"], key)
}

var environmentKeyListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS environment keys",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
	},
}

func environmentListKeyRun(v cli.Values) (cli.ListResult, error) {
	keys, err := client.EnvironmentKeysList(v[_ProjectKey], v["env-name"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(keys), nil
}

var environmentKeyDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS environment key",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
		{Name: "key-name"},
	},
}

func environmentDeleteKeyRun(v cli.Values) error {
	return client.EnvironmentKeysDelete(v[_ProjectKey], v["env-name"], v["key-name"])
}
