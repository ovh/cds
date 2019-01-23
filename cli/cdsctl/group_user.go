package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var groupUserCmd = cli.Command{
	Name:  "user",
	Short: "Manage CDS users group",
}

func groupUser() *cobra.Command {
	return cli.NewCommand(groupUserCmd, nil, []*cobra.Command{
		cli.NewListCommand(groupUserListCmd, groupUserListRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(groupUserAdd, groupUserAddRun, nil),
		cli.NewDeleteCommand(groupUserRemove, groupUserRemoveRun, nil),
		cli.NewCommand(groupUserSetAdmin, groupUserSetAdminRun, nil),
		cli.NewCommand(groupUserAdminRemove, groupUserAdminRemoveRun, nil),
	})
}

var groupUserListCmd = cli.Command{
	Name:  "list",
	Short: "List users into a group",
	Args: []cli.Arg{
		{Name: "groupname"},
	},
}

func groupUserListRun(v cli.Values) (cli.ListResult, error) {
	gr, err := client.GroupGet(v.GetString("groupname"))
	if err != nil {
		return nil, err
	}
	users := make([]sdk.User, 0, len(gr.Admins)+len(gr.Users))

	for _, admin := range gr.Admins {
		admin.GroupAdmin = true
		users = append(users, admin)
	}
	users = append(users, gr.Users...)

	return cli.AsListResult(users), nil
}

var groupUserAdd = cli.Command{
	Name:  "add",
	Short: "Add an user into a group",
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "username"},
	},
}

func groupUserAddRun(v cli.Values) error {
	return client.GroupUserAdd(v.GetString("groupname"), []string{v.GetString("username")})
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
	return client.GroupUserRemove(v.GetString("groupname"), v.GetString("username"))
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
	return client.GroupUserAdminSet(v.GetString("groupname"), v.GetString("username"))
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
	return client.GroupUserAdminRemove(v.GetString("groupname"), v.GetString("username"))
}
