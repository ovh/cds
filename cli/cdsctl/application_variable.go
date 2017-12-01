package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	applicationVariableCmd = cli.Command{
		Name:  "variable",
		Short: "Manage CDS application variables",
	}

	applicationVariable = cli.NewCommand(applicationVariableCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(applicationVariableCreateCmd, applicationCreateVariableRun, nil),
			cli.NewListCommand(applicationVariableListCmd, applicationListVariableRun, nil),
			cli.NewCommand(applicationVariableDeleteCmd, applicationDeleteVariableRun, nil),
			cli.NewCommand(applicationVariableUpdateCmd, applicationUpdateVariableRun, nil),
		})
)

var applicationVariableCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new variable on application. variable type can be one of password, text, string, key, boolean, number, repository",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "app-name"},
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func applicationCreateVariableRun(v cli.Values) error {
	variable := &sdk.Variable{
		Name:  v["variable-name"],
		Type:  v["variable-type"],
		Value: v["variable-value"],
	}
	return client.ApplicationVariableCreate(v["project-key"], v["app-name"], variable)
}

var applicationVariableListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS application variables",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "app-name"},
	},
}

func applicationListVariableRun(v cli.Values) (cli.ListResult, error) {
	variables, err := client.ApplicationVariablesList(v["project-key"], v["app-name"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(variables), nil
}

var applicationVariableDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS application variable",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "app-name"},
		{Name: "variable-name"},
	},
}

func applicationDeleteVariableRun(v cli.Values) error {
	return client.ApplicationVariableDelete(v["project-key"], v["app-name"], v["variable-name"])
}

var applicationVariableUpdateCmd = cli.Command{
	Name:  "update",
	Short: "Update CDS application variable value",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "app-name"},
		{Name: "variable-oldname"},
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func applicationUpdateVariableRun(v cli.Values) error {
	variable, err := client.ApplicationVariableGet(v["project-key"], v["app-name"], v["variable-oldname"])
	if err != nil {
		return err
	}
	variable.Name = v["variable-name"]
	variable.Value = v["variable-value"]
	variable.Type = v["variable-type"]
	return client.ApplicationVariableUpdate(v["project-key"], v["app-name"], variable)
}
