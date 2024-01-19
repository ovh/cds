package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectSecretCmd = cli.Command{
	Name:    "secret",
	Aliases: []string{"secrets"},
	Short:   "Manage secrets on a CDS project",
}

func projectSecret() *cobra.Command {
	return cli.NewCommand(projectSecretCmd, nil, []*cobra.Command{
		cli.NewCommand(projectSecretAddCmd, projectSecretAddFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(projectSecretListCmd, projectSecretListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(projectSecretDeleteCmd, projectSecretDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectSecretUpdateCmd, projectSecretUpdateFunc, nil, withAllCommandModifiers()...),
	})
}

var projectSecretListCmd = cli.Command{
	Name:  "list",
	Short: "List secrets available on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectSecretListFunc(v cli.Values) (cli.ListResult, error) {
	secrets, err := client.ProjectSecretList(context.Background(), v.GetString(_ProjectKey))
	return cli.AsListResult(secrets), err
}

var projectSecretDeleteCmd = cli.Command{
	Name:    "delete",
	Short:   "Delete a secret on a project",
	Aliases: []string{"remove", "rm"},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectSecretDeleteFunc(v cli.Values) error {
	return client.ProjectSecretDelete(context.Background(), v.GetString(_ProjectKey), v.GetString("name"))
}

var projectSecretAddCmd = cli.Command{
	Name:    "add",
	Short:   "Add a secret on a project",
	Example: "cdsctl project secret add MY-PROJECT <name> <value>",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
		{Name: "value"},
	},
}

func projectSecretAddFunc(v cli.Values) error {
	secret := sdk.ProjectSecret{
		Name:  v.GetString("name"),
		Value: v.GetString("value"),
	}

	return client.ProjectSecretAdd(context.Background(), v.GetString(_ProjectKey), secret)
}

var projectSecretUpdateCmd = cli.Command{
	Name:    "update",
	Short:   "Update a secret in a project",
	Example: "cdsctl project secret update MY-PROJECT <NAME> <VALUE>",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
		{Name: "value"},
	},
}

func projectSecretUpdateFunc(v cli.Values) error {
	secret := sdk.ProjectSecret{
		Name:  v.GetString("name"),
		Value: v.GetString("value"),
	}
	return client.ProjectSecretUpdate(context.Background(), v.GetString(_ProjectKey), secret)
}
