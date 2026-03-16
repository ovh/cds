package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminGroupCmd = cli.Command{
	Name:    "group",
	Aliases: []string{"groups"},
	Short:   "Manage CDS groups (admin only)",
}

func adminGroups() *cobra.Command {
	return cli.NewCommand(adminGroupCmd, nil, []*cobra.Command{
		cli.NewCommand(adminGroupCreateCmd, adminGroupCreateRun, nil, withAllCommandModifiers()...),
	})
}

var adminGroupCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS group with a specific first member",
	Args: []cli.Arg{
		{Name: "group-name"},
		{Name: "first-member-username"},
	},
}

func adminGroupCreateRun(v cli.Values) error {
	err := client.AdminGroupCreate(context.Background(), sdk.AdminCreateGroup{
		Name:                v.GetString("group-name"),
		FirstMemberUsername: v.GetString("first-member-username"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("Group %q created with %q as first member\n", v.GetString("group-name"), v.GetString("first-member-username"))
	return nil
}
