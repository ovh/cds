package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminUsersCmd = cli.Command{
	Name:    "users",
	Aliases: []string{"user"},
	Short:   "Manage CDS users",
}

func adminUsers() *cobra.Command {
	return cli.NewCommand(adminUsersCmd, nil, []*cobra.Command{
		cli.NewCommand(adminUserSetOrganizationCmd, adminUserSetOrganizationRun, nil),
		cli.NewCommand(adminUserRenameCmd, adminUserRenameRun, nil),
		cli.NewCommand(adminUserCreateCmd, adminUserCreateRun, nil),
	})
}

var adminUserCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a user",
	Args: []cli.Arg{
		{
			Name: "username",
		},
		{
			Name: "fullname",
		},
		{
			Name: "email",
		},
	},
}

func adminUserCreateRun(v cli.Values) error {
	ctx := context.Background()

	user := sdk.AuthConsumerSigninRequest{
		"username": v.GetString("username"),
		"fullname": v.GetString("fullname"),
		"email":    v.GetString("email"),
	}
	err := client.AdminUserCreate(ctx, user)
	if err != nil {
		return err
	}

	fmt.Println("User has been created")
	return nil
}

var adminUserRenameCmd = cli.Command{
	Name:  "rename",
	Short: "Rename a given user",
	Args: []cli.Arg{
		{
			Name: "username",
		},
		{
			Name: "new-username",
		},
	},
}

func adminUserRenameRun(v cli.Values) error {
	ctx := context.Background()
	username := v.GetString("username")
	usernameNew := v.GetString("new-username")

	u, err := client.UserGet(ctx, username)
	if err != nil {
		return err
	}
	u.Username = usernameNew
	if err := client.UserUpdate(ctx, username, u); err != nil {
		return err
	}

	fmt.Printf("User %q has been renamed to %q\n", username, usernameNew)
	return nil
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
