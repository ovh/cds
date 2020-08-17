package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectVariableCmd = cli.Command{
	Name:    "variable",
	Aliases: []string{"variables"},
	Short:   "Manage CDS project variables",
}

func projectVariable() *cobra.Command {
	return cli.NewCommand(projectVariableCmd, nil, []*cobra.Command{
		cli.NewCommand(projectVariableCreateCmd, projectCreateVariableRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(projectVariableListCmd, projectListVariableRun, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectVariableShowCmd, projectVariableShowRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectVariableDeleteCmd, projectDeleteVariableRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectVariableUpdateCmd, projectUpdateVariableRun, nil, withAllCommandModifiers()...),
	})
}

var projectVariableCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new variable on project. Variable type can be one of password, text, string, key, boolean, number, repository",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func projectCreateVariableRun(v cli.Values) error {
	variable := &sdk.Variable{
		Name:  v.GetString("variable-name"),
		Type:  v.GetString("variable-type"),
		Value: v.GetString("variable-value"),
	}
	return client.ProjectVariableCreate(v.GetString(_ProjectKey), variable)
}

var projectVariableListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS project variables",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectListVariableRun(v cli.Values) (cli.ListResult, error) {
	variables, err := client.ProjectVariablesList(v.GetString(_ProjectKey))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(variables), nil
}

var projectVariableDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS project variable",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variable-name"},
	},
}

func projectDeleteVariableRun(v cli.Values) error {
	return client.ProjectVariableDelete(v.GetString(_ProjectKey), v.GetString("variable-name"))
}

var projectVariableShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS project variable",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variable-name"},
	},
}

func projectVariableShowRun(v cli.Values) (interface{}, error) {
	return client.ProjectVariableGet(v.GetString(_ProjectKey), v.GetString("variable-name"))
}

var projectVariableUpdateCmd = cli.Command{
	Name:  "update",
	Short: "Update CDS project variable value",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variable-oldname"},
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func projectUpdateVariableRun(v cli.Values) error {
	variable, err := client.ProjectVariableGet(v.GetString(_ProjectKey), v.GetString("variable-oldname"))
	if err != nil {
		return err
	}
	variable.Name = v.GetString("variable-name")
	variable.Type = v.GetString("variable-type")
	variable.Value = v.GetString("variable-value")
	return client.ProjectVariableUpdate(v.GetString(_ProjectKey), variable)
}
