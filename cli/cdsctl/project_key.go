package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectKeyCmd = cli.Command{
	Name:    "keys",
	Aliases: []string{"key"},
	Short:   "Manage CDS project keys",
}

func projectKey() *cobra.Command {
	return cli.NewCommand(projectKeyCmd, nil, []*cobra.Command{
		cli.NewCommand(projectKeyCreateCmd, projectCreateKeyRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(projectKeyListCmd, projectListKeyRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectKeyDeleteCmd, projectDeleteKeyRun, nil, withAllCommandModifiers()...),
	})
}

var projectKeyCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new key on project. key-type can be ssh or pgp",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "key-name"},
		{Name: "key-type"},
	},
}

func projectCreateKeyRun(v cli.Values) error {
	key := &sdk.ProjectKey{
		Name: v.GetString("key-name"),
		Type: sdk.KeyType(v.GetString("key-type")),
	}
	if err := client.ProjectKeyCreate(v.GetString(_ProjectKey), key); err != nil {
		return err
	}

	fmt.Printf("Project key %s of type %s created with success in project %s\n", key.Name, key.Type, v.GetString(_ProjectKey))
	fmt.Println(key.Public)
	return nil
}

var projectKeyListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS project keys",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectListKeyRun(v cli.Values) (cli.ListResult, error) {
	keys, err := client.ProjectKeysList(v.GetString(_ProjectKey))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(keys), nil
}

var projectKeyDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS project key",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "key-name"},
	},
}

func projectDeleteKeyRun(v cli.Values) error {
	return client.ProjectKeysDelete(v.GetString(_ProjectKey), v.GetString("key-name"))
}
