package main

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var projectVariableSetCmd = cli.Command{
	Name:    "variableset",
	Aliases: []string{"vs"},
	Short:   "Manage Variable Set on a CDS project",
}

func projectVariableSet() *cobra.Command {
	return cli.NewCommand(projectVariableSetCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectVariableSetListCmd, projectVariableSetListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(projectVariableSetDeleteCmd, projectVariableSetDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectVariableSetCreateCmd, projectVariableSetCreateFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectVariableSetShowCmd, projectVariableSetShowFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectVariableSetCreateFromApplicationCmd, projectVariableSetCreateFromApplicationFunc, nil, withAllCommandModifiers()...),
		projectVariableSetItem(),
	})
}

var projectVariableSetShowCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},

	Short: "Get the given variable set",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectVariableSetShowFunc(v cli.Values) (interface{}, error) {
	vs, err := client.ProjectVariableSetShow(context.Background(), v.GetString(_ProjectKey), v.GetString("name"))
	return vs, err
}

var projectVariableSetListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List all variable sets in the given project",
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
	Short:   "Delete a variable set on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
	Flags: []cli.Flag{
		{
			Name: "with-items",
			Type: cli.FlagBool,
		},
	},
}

func projectVariableSetDeleteFunc(v cli.Values) error {
	mod := cdsclient.WithQueryParameter("force", strconv.FormatBool(v.GetBool("with-items")))
	return client.ProjectVariableSetDelete(context.Background(), v.GetString(_ProjectKey), v.GetString("name"), mod)
}

var projectVariableSetCreateFromApplicationCmd = cli.Command{
	Name:    "from-application",
	Aliases: []string{"fa"},
	Short:   "Create a new variableset inside the given project",
	Example: "cdsctl exp project variableset from-application MY-PROJECT MY-VARIABLESET-NAME MY-APPLICATION",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
		{Name: "application-name"},
	},
}

func projectVariableSetCreateFromApplicationFunc(v cli.Values) error {
	copyReq := sdk.CopyApplicationVariableToVariableSet{
		ApplicationName: v.GetString("application-name"),
		VariableSetName: v.GetString("name"),
	}

	return client.ProjectVariableSetCreateFromApplication(context.Background(), v.GetString(_ProjectKey), copyReq)
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

	return client.ProjectVariableSetCreate(context.Background(), v.GetString(_ProjectKey), &vs)
}
