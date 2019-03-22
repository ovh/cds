package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var applicationVariableCmd = cli.Command{
	Name:  "variable",
	Short: "Manage CDS application variables",
}

func applicationVariable() *cobra.Command {
	return cli.NewCommand(applicationVariableCmd, nil, []*cobra.Command{
		cli.NewCommand(applicationVariableCreateCmd, applicationCreateVariableRun, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(applicationVariableShowCmd, applicationVariableShowRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(applicationVariableListCmd, applicationListVariableRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(applicationVariableDeleteCmd, applicationDeleteVariableRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(applicationVariableUpdateCmd, applicationUpdateVariableRun, nil, withAllCommandModifiers()...),
	})
}

var applicationVariableCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new variable on application. variable type can be one of password, text, string, key, boolean, number, repository",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
	Args: []cli.Arg{
		{Name: "variable-name", Weight: 1},
		{Name: "variable-type", Weight: 2},
		{Name: "variable-value", Weight: 3},
	},
}

func applicationCreateVariableRun(v cli.Values) error {
	variable := &sdk.Variable{
		Name:  v.GetString("variable-name"),
		Type:  v.GetString("variable-type"),
		Value: v.GetString("variable-value"),
	}
	return client.ApplicationVariableCreate(v.GetString(_ProjectKey), v.GetString(_ApplicationName), variable)
}

var applicationVariableListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS application variables",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
}

func applicationListVariableRun(v cli.Values) (cli.ListResult, error) {
	variables, err := client.ApplicationVariablesList(v.GetString(_ProjectKey), v.GetString(_ApplicationName))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(variables), nil
}

var applicationVariableDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS application variable",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
	Args: []cli.Arg{
		{Name: "variable-name"},
	},
}

func applicationDeleteVariableRun(v cli.Values) error {
	return client.ApplicationVariableDelete(v.GetString(_ProjectKey), v.GetString(_ApplicationName), v.GetString("variable-name"))
}

var applicationVariableShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS application variable",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
	Args: []cli.Arg{
		{Name: "variable-name"},
	},
}

func applicationVariableShowRun(v cli.Values) (interface{}, error) {
	return client.ApplicationVariableGet(v.GetString(_ProjectKey), v.GetString(_ApplicationName), v.GetString("variable-name"))
}

var applicationVariableUpdateCmd = cli.Command{
	Name:  "update",
	Short: "Update CDS application variable value",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
	Args: []cli.Arg{
		{Name: "variable-oldname", Weight: 1},
		{Name: "variable-name", Weight: 2},
		{Name: "variable-type", Weight: 3},
		{Name: "variable-value", Weight: 4},
	},
}

func applicationUpdateVariableRun(v cli.Values) error {
	variable, err := client.ApplicationVariableGet(v.GetString(_ProjectKey), v.GetString(_ApplicationName), v.GetString("variable-oldname"))
	if err != nil {
		return err
	}
	variable.Name = v.GetString("variable-name")
	variable.Value = v.GetString("variable-value")
	variable.Type = v.GetString("variable-type")
	return client.ApplicationVariableUpdate(v.GetString(_ProjectKey), v.GetString(_ApplicationName), variable)
}
