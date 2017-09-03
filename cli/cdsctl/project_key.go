package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	projectKeyCmd = cli.Command{
		Name:  "keys",
		Short: "Manage CDS project keys",
	}

	projectKey = cli.NewCommand(projectKeyCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(projectKeyCreateCmd, projectCreateKeyRun, nil),
			cli.NewListCommand(projectKeyListCmd, projectListKeyRun, nil),
			cli.NewCommand(projectKeyDeleteCmd, projectDeleteKeyRun, nil),
		})
)

var projectKeyCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new key on project. key type can be ssh or pgp",
	Args: []cli.Arg{
		{Name: "key"},
		{Name: "keyName"},
		{Name: "keyType"},
	},
}

func projectCreateKeyRun(v cli.Values) error {
	key := &sdk.ProjectKey{
		Key: sdk.Key{
			Name: v["keyName"],
			Type: v["keyType"],
		},
	}
	if err := client.ProjectKeyCreate(v["key"], key); err != nil {
		return err
	}
	return nil
}

var projectKeyListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS project keys",
	Args: []cli.Arg{
		{Name: "key"},
	},
}

func projectListKeyRun(v cli.Values) (cli.ListResult, error) {
	keys, err := client.ProjectKeysList(v["key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(keys), nil
}

var projectKeyDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS project key",
	Args: []cli.Arg{
		{Name: "key"},
		{Name: "keyName"},
	},
}

func projectDeleteKeyRun(v cli.Values) error {
	return client.ProjectKeysDelete(v["key"], v["keyName"])
}
