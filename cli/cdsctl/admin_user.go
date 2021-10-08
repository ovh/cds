package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var adminUsersCmd = cli.Command{
	Name:    "users",
	Aliases: []string{"user"},
	Short:   "Manage CDS users",
}

func adminUsers() *cobra.Command {
	return cli.NewCommand(adminUsersCmd, nil, []*cobra.Command{
		cli.NewCommand(adminUserSetOrganizationCmd, adminUserSetOrganizationRun, nil),
	})
}

var adminUserSetOrganizationCmd = cli.Command{
	Name:  "set-organization",
	Short: "Set organization for given user",
	Args: []cli.Arg{
		{
			Name: "username",
		},
		{
			Name: "organization",
		},
	},
}

func adminUserSetOrganizationRun(v cli.Values) error {
	ctx := context.Background()
	username := v.GetString("username")
	organization := v.GetString("organization")

	u, err := client.UserGet(ctx, username)
	if err != nil {
		return err
	}
	if u.Organization != "" {
		return cli.NewError("user organization already set to %q", u.Organization)
	}

	u.Organization = organization

	if err := client.UserUpdate(ctx, u.Username, u); err != nil {
		return err
	}

	fmt.Printf("User organization set to %q\n", u.Organization)
	return nil
}
