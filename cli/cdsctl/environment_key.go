package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var environmentKeyCmd = cli.Command{
	Name:    "keys",
	Aliases: []string{"key"},
	Short:   "Manage CDS environment keys",
}

func environmentKey() *cobra.Command {
	return cli.NewCommand(environmentKeyCmd, nil, []*cobra.Command{
		cli.NewCommand(environmentKeyCreateCmd, environmentCreateKeyRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(environmentKeyListCmd, environmentListKeyRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(environmentKeyDeleteCmd, environmentDeleteKeyRun, nil, withAllCommandModifiers()...),
	})
}

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
		Name: v.GetString("key-name"),
		Type: sdk.KeyType(v.GetString("key-type")),
	}
	if err := client.EnvironmentKeyCreate(v.GetString(_ProjectKey), v.GetString("env-name"), key); err != nil {
		return err
	}

	fmt.Printf("Environment key %s of type %s created with success in environment %s\n", key.Name, key.Type, v.GetString("env-name"))
	fmt.Println(key.Public)
	return nil
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
	keys, err := client.EnvironmentKeysList(v.GetString(_ProjectKey), v.GetString("env-name"))
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
	return client.EnvironmentKeysDelete(v.GetString(_ProjectKey), v.GetString("env-name"), v.GetString("key-name"))
}
