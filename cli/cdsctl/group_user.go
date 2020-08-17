package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var groupMemberCmd = cli.Command{
	Name:    "member",
	Aliases: []string{"members"},
	Short:   "Manage group's member",
}

func groupMember() *cobra.Command {
	return cli.NewCommand(groupMemberCmd, nil, []*cobra.Command{
		cli.NewListCommand(groupMemberListCmd, groupMemberListRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(groupMemberAddCmd, groupMemberAddRun, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(groupMemberRemoveCmd, groupMemberRemoveRun, nil, withAllCommandModifiers()...),
	})
}

var groupMemberListCmd = cli.Command{
	Name:  "list",
	Short: "List members into a group",
	Args: []cli.Arg{
		{Name: "group-name"},
	},
}

func groupMemberListRun(v cli.Values) (cli.ListResult, error) {
	gr, err := client.GroupGet(v.GetString("group-name"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(gr.Members), nil
}

var groupMemberAddCmd = cli.Command{
	Name:  "add",
	Short: "Add or edit a member for a group",
	Args: []cli.Arg{
		{Name: "group-name"},
		{Name: "username"},
		{
			Name: "admin",
			IsValid: func(admin string) bool {
				return admin == sdk.TrueString || admin == sdk.FalseString
			},
		},
	},
}

func groupMemberAddRun(v cli.Values) error {
	gr, err := client.GroupGet(v.GetString("group-name"))
	if err != nil {
		return err
	}

	username := v.GetString("username")
	admin := v.GetString("admin") == sdk.TrueString
	var member *sdk.GroupMember
	for i := range gr.Members {
		if gr.Members[i].Username == username {
			member = &gr.Members[i]
			break
		}
	}

	if member == nil {
		_, err = client.GroupMemberAdd(v.GetString("group-name"), &sdk.GroupMember{
			Username: v.GetString("username"),
			Admin:    v.GetString("admin") == sdk.TrueString,
		})
	} else if member.Admin != admin {
		_, err = client.GroupMemberEdit(v.GetString("group-name"), &sdk.GroupMember{
			Username: v.GetString("username"),
			Admin:    admin,
		})
	}

	return err
}

var groupMemberRemoveCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a member of a group",
	Args: []cli.Arg{
		{Name: "group-name"},
		{Name: "username"},
	},
}

func groupMemberRemoveRun(v cli.Values) error {
	return client.GroupMemberRemove(v.GetString("group-name"), v.GetString("username"))
}
