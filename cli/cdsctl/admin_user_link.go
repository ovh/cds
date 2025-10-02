package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminUserLinkCmd = cli.Command{
	Name:  "link",
	Short: "Manage user <-> external identity links",
}

func adminUserLink() *cobra.Command {
	return cli.NewCommand(adminUserLinkCmd, nil, []*cobra.Command{
		cli.NewCommand(adminUserLinkCreateCmd, adminUserLinkCreateRun, nil),
		cli.NewCommand(adminUserLinkDeleteCmd, adminUserLinkDeleteRun, nil),
	})
}

var adminUserLinkCreateCmd = cli.Command{
	Name:    "create",
	Aliases: []string{"add"},
	Short:   "Create a link between a user and an external identity (consumer)",
	Args: []cli.Arg{
		{Name: "username"},
		{Name: "consumerType"},
		{Name: "externalUsername"},
	},
}

func adminUserLinkCreateRun(v cli.Values) error {
	ctx := context.Background()
	username := v.GetString("username")

	link := sdk.UserLink{
		Type:       v.GetString("consumerType"),
		ExternalID: v.GetString("externalUsername"),
		Username:   v.GetString("externalUsername"),
	}

	if link.Type != string(sdk.ConsumerBitbucketServer) {
		return fmt.Errorf("unsupported consumer type: %s. Only %s is managed", link.Type, string(sdk.ConsumerBitbucketServer))
	}

	if err := client.AdminUserLinkCreate(ctx, username, link); err != nil {
		return err
	}
	fmt.Printf("Link created (user=%s type=%s externalUsername=%s)\n", username, link.Type, link.Username)
	return nil
}

var adminUserLinkDeleteCmd = cli.Command{
	Name:    "delete",
	Short:   "Delete a link between a user and an external identity (consumer)",
	Aliases: []string{"remove", "rm"},
	Args: []cli.Arg{
		{Name: "username"},
		{Name: "consumerType"},
	},
}

func adminUserLinkDeleteRun(v cli.Values) error {
	ctx := context.Background()
	username := v.GetString("username")

	links, err := client.UserLinks(ctx, username)
	if err != nil {
		return err
	}

	var link *sdk.UserLink
	for _, l := range links {
		if l.Type == v.GetString("consumerType") {
			link = &l
			break
		}
	}
	if link == nil {
		return fmt.Errorf("no link found for user %s with type %s", username, v.GetString("consumerType"))
	}

	if err := client.AdminUserLinkDelete(ctx, username, *link); err != nil {
		return err
	}
	fmt.Printf("Link deleted (user=%s type=%s)\n", username, link.Type)
	return nil
}
