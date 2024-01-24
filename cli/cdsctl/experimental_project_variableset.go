package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectVariableSetCmd = cli.Command{
	Name:    "variableset",
	Aliases: []string{"vs"},
	Short:   "Manage VariableSet on a CDS project",
}

func projectVariableSet() *cobra.Command {
	return cli.NewCommand(projectVariableSetCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectVariableSetListCmd, projectVariableSetListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(projectVariableSetDeleteCmd, projectVariableSetDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectVariableSetCreateCmd, projectVariableSetCreateFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(projectVariableSetShowCmd, projectVariableSetShowFunc, nil, withAllCommandModifiers()...),
	})
}

var projectVariableSetShowCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},

	Short: "Get the given variableset",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectVariableSetShowFunc(v cli.Values) (cli.ListResult, error) {
	vs, err := client.ProjectVariableSetShow(context.Background(), v.GetString(_ProjectKey), v.GetString("name"))
	return cli.AsListResult(vs.Items), err
}

var projectVariableSetListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List all variabelset in the given project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectVariableSetListFunc(v cli.Values) (cli.ListResult, error) {
	vss, err := client.ProjectVariableSetList(context.Background(), v.GetString(_ProjectKey))
	return cli.AsListResult(vss), err
}

var projectVariableSetDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a variableset on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectVariableSetDeleteFunc(v cli.Values) error {
	return client.ProjectVariableSetDelete(context.Background(), v.GetString(_ProjectKey), v.GetString("name"))
}

var projectVariableSetCreateCmd = cli.Command{
	Name:    "add",
	Aliases: []string{"create"},
	Short:   "Create a new variableset inside the given project",
	Example: "cdsctl exp project variableset add MY-PROJECT MY-VARIABLESET-NAME",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectVariableSetCreateFunc(v cli.Values) error {
	vs := sdk.ProjectVariableSet{
		Name: v.GetString("name"),
	}
	if err := client.ProjectVariableSetCreate(context.Background(), v.GetString(_ProjectKey), &vs); err != nil {
		return cli.WrapError(err, "unable to get notification")
	}
	return nil
}
