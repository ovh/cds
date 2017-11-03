package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
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
