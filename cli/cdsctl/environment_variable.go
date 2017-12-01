package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	environmentVariableCmd = cli.Command{
		Name:  "variable",
		Short: "Manage CDS environment variables",
	}

	environmentVariable = cli.NewCommand(environmentVariableCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(environmentVariableCreateCmd, environmentCreateVariableRun, nil),
			cli.NewListCommand(environmentVariableListCmd, environmentListVariableRun, nil),
			cli.NewCommand(environmentVariableDeleteCmd, environmentDeleteVariableRun, nil),
			cli.NewCommand(environmentVariableUpdateCmd, environmentUpdateVariableRun, nil),
		})
)

var environmentVariableCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new variable on environment. variable type can be one of password, text, string, key, boolean, number, repository",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "env-name"},
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func environmentCreateVariableRun(v cli.Values) error {
	variable := &sdk.Variable{
		Name:  v["variable-name"],
		Type:  v["variable-type"],
		Value: v["variable-value"],
	}
	return client.EnvironmentVariableCreate(v["project-key"], v["env-name"], variable)
}

var environmentVariableListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS environment variables",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "env-name"},
	},
}

func environmentListVariableRun(v cli.Values) (cli.ListResult, error) {
	variables, err := client.EnvironmentVariablesList(v["project-key"], v["env-name"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(variables), nil
}

var environmentVariableDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS environment variable",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "env-name"},
		{Name: "variable-name"},
	},
}

func environmentDeleteVariableRun(v cli.Values) error {
	return client.EnvironmentVariableDelete(v["project-key"], v["env-name"], v["variable-name"])
}

var environmentVariableUpdateCmd = cli.Command{
	Name:  "update",
	Short: "Update CDS environment variable value",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "env-name"},
		{Name: "variable-oldname"},
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func environmentUpdateVariableRun(v cli.Values) error {
	variable, err := client.EnvironmentVariableGet(v["project-key"], v["env-name"], v["variable-oldname"])
	if err != nil {
		return err
	}
	variable.Name = v["variable-name"]
	variable.Value = v["variable-value"]
	variable.Type = v["variable-type"]
	return client.EnvironmentVariableUpdate(v["project-key"], v["env-name"], variable)
}
