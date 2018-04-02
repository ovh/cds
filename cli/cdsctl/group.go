package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
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
			cli.NewCommand(groupCreateCmd, groupCreateRun, nil),
			cli.NewCommand(groupRenameCmd, groupRenameRun, nil),
			cli.NewDeleteCommand(groupDeleteCmd, groupDeleteRun, nil),
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
		{Name: "group-name"},
	},
}

func groupShowRun(v cli.Values) (interface{}, error) {
	group, err := client.GroupGet(v["group-name"])
	if err != nil {
		return nil, err
	}
	return *group, nil
}

var groupCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS group",
	Args: []cli.Arg{
		{Name: "group-name"},
	},
	Aliases: []string{"add"},
}

func groupCreateRun(v cli.Values) error {
	gr := &sdk.Group{Name: v["group-name"]}
	return client.GroupCreate(gr)
}

var groupRenameCmd = cli.Command{
	Name:  "rename",
	Short: "Rename a CDS group",
	Args: []cli.Arg{
		{Name: "old-group-name"},
		{Name: "new-group-name"},
	},
}

func groupRenameRun(v cli.Values) error {
	return client.GroupRename(v["old-group-name"], v["new-group-name"])
}

var groupDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS group",
	Args: []cli.Arg{
		{Name: "group-name"},
	},
}

func groupDeleteRun(v cli.Values) error {
	err := client.GroupDelete(v["group-name"])
	if err != nil {
		if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrGroupNotFound) {
			fmt.Println(err.Error())
			return nil
		}
	}

	return err
}
