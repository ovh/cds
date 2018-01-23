package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	projectVariableCmd = cli.Command{
		Name:  "variable",
		Short: "Manage CDS project variables",
	}

	projectVariable = cli.NewCommand(projectVariableCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(projectVariableCreateCmd, projectCreateVariableRun, nil, withAllCommandModifiers()...),
			cli.NewListCommand(projectVariableListCmd, projectListVariableRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(projectVariableDeleteCmd, projectDeleteVariableRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(projectVariableUpdateCmd, projectUpdateVariableRun, nil, withAllCommandModifiers()...),
		})
)

var projectVariableCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new variable on project. Variable type can be one of password, text, string, key, boolean, number, repository",
	Ctx: []cli.Arg{
		{Name: "project-key"},
	},
	Args: []cli.Arg{
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func projectCreateVariableRun(v cli.Values) error {
	variable := &sdk.Variable{
		Name:  v["variable-name"],
		Type:  v["variable-type"],
		Value: v["variable-value"],
	}
	return client.ProjectVariableCreate(v["project-key"], variable)
}

var projectVariableListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS project variables",
	Ctx: []cli.Arg{
		{Name: "project-key"},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
	},
}

func projectListVariableRun(v cli.Values) (cli.ListResult, error) {
	variables, err := client.ProjectVariablesList(v["project-key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(variables), nil
}

var projectVariableDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS project variable",
	Ctx: []cli.Arg{
		{Name: "project-key"},
	},
	Args: []cli.Arg{
		{Name: "env-name"},
		{Name: "variable-name"},
	},
}

func projectDeleteVariableRun(v cli.Values) error {
	return client.ProjectVariableDelete(v["project-key"], v["variable-name"])
}

var projectVariableUpdateCmd = cli.Command{
	Name:  "update",
	Short: "Update CDS project variable value",
	Ctx: []cli.Arg{
		{Name: "project-key"},
	},
	Args: []cli.Arg{
		{Name: "variable-oldname"},
		{Name: "variable-name"},
		{Name: "variable-type"},
		{Name: "variable-value"},
	},
}

func projectUpdateVariableRun(v cli.Values) error {
	variable, err := client.ProjectVariableGet(v["project-key"], v["variable-oldname"])
	if err != nil {
		return err
	}
	variable.Name = v["variable-name"]
	variable.Value = v["variable-value"]
	variable.Type = v["variable-type"]
	return client.ProjectVariableUpdate(v["project-key"], variable)
}
