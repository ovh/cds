package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	groupCmd = cli.Command{
		Name:  "group",
		Short: "Manage CDS group",
	}

	group = cli.NewCommand(groupCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(groupListCmd, groupListRun, nil),
			cli.NewGetCommand(groupShowCmd, groupShowRun, nil),
			groupToken,
			groupUser,
		})
)

var groupListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS groups",
}

func groupListRun(v cli.Values) (cli.ListResult, error) {
	apps, err := client.GroupList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(apps), nil
}

var groupShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS group",
	Args: []cli.Arg{
		{Name: "groupname"},
	},
}

func groupShowRun(v cli.Values) (interface{}, error) {
	group, err := client.GroupGet(v["groupname"])
	if err != nil {
		return nil, err
	}
	return *group, nil
}
