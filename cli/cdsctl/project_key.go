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
			cli.NewCommand(projectKeyCreateCmd, projectCreateKeyRun, nil, withAllCommandModifiers()...),
			cli.NewListCommand(projectKeyListCmd, projectListKeyRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(projectKeyDeleteCmd, projectDeleteKeyRun, nil, withAllCommandModifiers()...),
		})
)

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
		Key: sdk.Key{
			Name: v["key-name"],
			Type: v["key-type"],
		},
	}
	return client.ProjectKeyCreate(v[_ProjectKey], key)
}

var projectKeyListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS project keys",
	Args: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectListKeyRun(v cli.Values) (cli.ListResult, error) {
	keys, err := client.ProjectKeysList(v[_ProjectKey])
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
	return client.ProjectKeysDelete(v[_ProjectKey], v["key-name"])
}
