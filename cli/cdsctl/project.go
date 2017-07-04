package main

import (
	"github.com/ovh/cds/cli"
	"github.com/spf13/cobra"
)

var (
	projectCmd = cli.Command{
		Name:  "project",
		Short: "Manage CDS project",
	}

	project = cli.NewCommand(projectCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(projectListCmd, projectListRun, nil),
		})
)

var projectListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS projects",
}

func projectListRun(v cli.Values) (cli.ListResult, error) {
	projs, err := client.ProjectList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(projs), nil
}
