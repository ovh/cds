package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var environmentVariableCmd = cli.Command{
	Name:    "variable",
	Aliases: []string{"variables"},
	Short:   "Manage CDS environment variables",
}

func environmentVariable() *cobra.Command {
	return cli.NewCommand(environmentVariableCmd, nil, []*cobra.Command{
		cli.NewCommand(environmentVariableCreateCmd, environmentCreateVariableRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(environmentVariableListCmd, environmentListVariableRun, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(environmentVariableShowCmd, environmentVariableShowRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(environmentVariableDeleteCmd, environmentDeleteVariableRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(environmentVariableUpdateCmd, environmentUpdateVariableRun, nil, withAllCommandModifiers()...),
	})
}

var environmentVariableCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new variable on environment. variable type can be one of password, text, string, key, boolean, number, repository",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func environmentCreateVariableRun(v cli.Values) error {
	variable := &sdk.Variable{
		Name:  v.GetString("variable-name"),
		Type:  v.GetString("variable-type"),
		Value: v.GetString("variable-value"),
	}
	return client.EnvironmentVariableCreate(v.GetString(_ProjectKey), v.GetString("env-name"), variable)
}

var environmentVariableListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS environment variables",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
	},
}

func environmentListVariableRun(v cli.Values) (cli.ListResult, error) {
	variables, err := client.EnvironmentVariablesList(v.GetString(_ProjectKey), v.GetString("env-name"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(variables), nil
}

var environmentVariableDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS environment variable",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
		{Name: "variable-name"},
	},
}

func environmentDeleteVariableRun(v cli.Values) error {
	return client.EnvironmentVariableDelete(v.GetString(_ProjectKey), v.GetString("env-name"), v.GetString("variable-name"))
}

var environmentVariableShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS environment variable",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
		{Name: "variable-name"},
	},
}

func environmentVariableShowRun(v cli.Values) (interface{}, error) {
	return client.EnvironmentVariableGet(v.GetString(_ProjectKey), v.GetString("env-name"), v.GetString("variable-name"))
}

var environmentVariableUpdateCmd = cli.Command{
	Name:  "update",
	Short: "Update CDS environment variable value",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
		{Name: "variable-oldname"},
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func environmentUpdateVariableRun(v cli.Values) error {
	variable, err := client.EnvironmentVariableGet(v.GetString(_ProjectKey), v.GetString("env-name"), v.GetString("variable-oldname"))
	if err != nil {
		return err
	}
	variable.Name = v.GetString("variable-name")
	variable.Value = v.GetString("variable-value")
	variable.Type = v.GetString("variable-type")
	return client.EnvironmentVariableUpdate(v.GetString(_ProjectKey), v.GetString("env-name"), variable)
}
