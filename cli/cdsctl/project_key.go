package main

import (
	"github.com/ovh/cds/cli"
	"github.com/spf13/cobra"
)

var (
	projectKeyCmd = cli.Command{
		Name:  "key",
		Short: "Manage CDS project key",
	}

	projectKey = cli.NewCommand(projectKeyCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(projectKeyListCmd, projectListKeyRun, nil),
		})
)

var projectKeyListCmd = cli.Command{
	Name:  "list key",
	Short: "List CDS project keys",
	Args: []cli.Arg{
		{Name: "key"},
	},
}

func projectListKeyRun(v cli.Values) (cli.ListResult, error) {
	keys, err := client.ProjectListKeys(v["key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(keys), nil
}
