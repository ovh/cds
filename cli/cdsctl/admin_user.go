package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var adminUsersCmd = cli.Command{
	Name:    "users",
	Aliases: []string{"user"},
	Short:   "Manage CDS users",
}

func adminUsers() *cobra.Command {
	return cli.NewCommand(adminUsersCmd, nil, []*cobra.Command{
		cli.NewCommand(adminUserSetOrganizationCmd, adminUserSetOrganizationRun, nil),
		cli.NewCommand(adminUserSetEmailCmd, adminUserSetEmailRun, nil),
		cli.NewCommand(adminUserRenameCmd, adminUserRenameRun, nil),
		cli.NewCommand(adminUserCreateCmd, adminUserCreateRun, nil),
		cli.NewListCommand(adminUserSearchCmd, adminUserSearchRun, nil),
		cli.NewDeleteCommand(adminUserDeleteCmd, adminUserDeleteRun, nil),
		adminUserLink(),
		adminUserGroupList(),
		adminUserConsumer(),
	})
}

var adminUserSearchCmd = cli.Command{
	Name:  "search",
	Short: "Search CDS users by external link (consumer type + external username)",
	Flags: []cli.Flag{
		{
			Name:      "consumer-type",
			Usage:     "External consumer type (e.g. forgejo, bitbucketserver)",
			ShortHand: "t",
		},
		{
			Name:      "external-username",
			Usage:     "Username on the external consumer",
			ShortHand: "u",
		},
	},
}

func adminUserSearchRun(v cli.Values) (cli.ListResult, error) {
	ctx := context.Background()
	filter := &cdsclient.AdminUserFilter{
		ConsumerType:     v.GetString("consumer-type"),
		ExternalUsername: v.GetString("external-username"),
	}
	users, err := client.AdminUserSearch(ctx, filter)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(users), nil
}

func adminUserGroupList() *cobra.Command {
	return cli.NewListCommand(
		cli.Command{
			Name:  "group-list",
			Short: "List groups of a given user",
			Args: []cli.Arg{
				{Name: "username"},
			},
		},
		adminUserGroupListRun,
		nil,
	)
}

func adminUserGroupListRun(v cli.Values) (cli.ListResult, error) {
	ctx := context.Background()
	users, err := client.UserGetGroups(ctx, v.GetString("username"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(users), nil
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
		{
			Name: "organization",
		},
	},
}

func adminUserCreateRun(v cli.Values) error {
	ctx := context.Background()

	user := sdk.CreateUser{
		Username:     v.GetString("username"),
		Fullname:     v.GetString("fullname"),
		Email:        v.GetString("email"),
		Organization: v.GetString("organization"),
	}
	err := client.AdminUserCreate(ctx, user)
	if err != nil {
		return err
	}

	fmt.Println("User has been created")
	return nil
}

var adminUserDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a user",
	Args: []cli.Arg{
		{Name: "username"},
	},
}

func adminUserDeleteRun(v cli.Values) error {
	return client.UserDelete(context.Background(), v.GetString("username"))
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

var adminUserSetEmailCmd = cli.Command{
	Name:  "set-email",
	Short: "Set the primary email for a given user",
	Args: []cli.Arg{
		{
			Name: "username",
		},
		{
			Name: "email",
		},
	},
}

func adminUserSetEmailRun(v cli.Values) error {
	ctx := context.Background()
	username := v.GetString("username")
	email := v.GetString("email")

	contact := sdk.UserContact{
		Type:  sdk.UserContactTypeEmail,
		Value: email,
	}
	if err := client.AdminUserSetContact(ctx, username, contact); err != nil {
		return err
	}

	fmt.Printf("User %q email set to %q\n", username, email)
	return nil
}
