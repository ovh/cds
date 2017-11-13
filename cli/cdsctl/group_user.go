package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	groupUserCmd = cli.Command{
		Name:  "user",
		Short: "Manage CDS users group",
	}

	groupUser = cli.NewCommand(groupUserCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(groupUserAdd, groupUserAddRun, nil),
			cli.NewDeleteCommand(groupUserRemove, groupUserRemoveRun, nil),
			cli.NewCommand(groupUserSetAdmin, groupUserSetAdminRun, nil),
			cli.NewCommand(groupUserAdminRemove, groupUserAdminRemoveRun, nil),
		})
)

var groupUserAdd = cli.Command{
	Name:  "add",
	Short: "Add a user into a group",
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "username"},
	},
}

func groupUserAddRun(v cli.Values) error {
	return client.GroupUserAdd(v["groupname"], []string{v["username"]})
}

var groupUserRemove = cli.Command{
	Name:  "delete",
	Short: "Delete an user from a group",
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "username"},
	},
}

func groupUserRemoveRun(v cli.Values) error {
	return client.GroupUserRemove(v["groupname"], v["username"])
}

var groupUserSetAdmin = cli.Command{
	Name:  "setAdmin",
	Short: "Set a user as an administrator of a group",
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "username"},
	},
}

func groupUserSetAdminRun(v cli.Values) error {
	return client.GroupUserAdminSet(v["groupname"], v["username"])
}

var groupUserAdminRemove = cli.Command{
	Name:  "removeAdmin",
	Short: "Remove a user from administrators of a group",
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "username"},
	},
}

func groupUserAdminRemoveRun(v cli.Values) error {
	return client.GroupUserAdminRemove(v["groupname"], v["username"])
}
