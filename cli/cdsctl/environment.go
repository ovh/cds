package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	environmentCmd = cli.Command{
		Name:  "environment",
		Short: "Manage CDS environment",
		Aliases: []string{
			"env",
		},
	}

	environment = cli.NewCommand(environmentCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(environmentListCmd, environmentListRun, nil),
			cli.NewCommand(environmentCreateCmd, environmentCreateRun, nil),
			environmentKey,
			environmentVariable,
			environmentGroup,
		})
)

var environmentListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS environments",
	Args: []cli.Arg{
		{Name: "project-key"},
	},
}

func environmentListRun(v cli.Values) (cli.ListResult, error) {
	apps, err := client.EnvironmentList(v["project-key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(apps), nil
}

var environmentCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS environment",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "environment-name"},
	},
	Aliases: []string{"add"},
}

func environmentCreateRun(v cli.Values) error {
	env := &sdk.Environment{Name: v["environment-name"]}
	return client.EnvironmentCreate(v["project-key"], env)
}
