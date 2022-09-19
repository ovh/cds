package main

import (
	"context"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminOrganizationCmd = cli.Command{
	Name:    "organization",
	Aliases: []string{"org", "orga"},
	Short:   "Manage CDS CDN uService",
}

func adminOrganization() *cobra.Command {
	return cli.NewCommand(adminOrganizationCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminOrganizationListCmd, adminOrganizationListRun, nil),
		cli.NewCommand(adminOrganizationAddCmd, adminOrganizationAddRun, nil),
		cli.NewCommand(adminOrganizationDeleteCmd, adminOrganizationDeleteRun, nil),
		cli.NewCommand(adminOrganizationUserMigrateCmd, adminOrganizationUserMigrateRun, nil),
	})
}

var adminOrganizationAddCmd = cli.Command{
	Name:    "add",
	Short:   "Add a new Organization on CDS",
	Example: "cdsctl admin organization add <organization-name>",
	Args: []cli.Arg{
		{
			Name: "organization-name",
		},
	},
}

func adminOrganizationAddRun(v cli.Values) error {
	if err := client.AdminOrganizationCreate(context.Background(), sdk.Organization{Name: v.GetString("organization-name")}); err != nil {
		return err
	}
	return nil
}

var adminOrganizationListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List all organizations",
	Example: "cdsctl admin organization list",
}

func adminOrganizationListRun(_ cli.Values) (cli.ListResult, error) {
	orgs, err := client.AdminOrganizationList(context.Background())
	return cli.AsListResult(orgs), err
}

var adminOrganizationDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete an organization",
	Example: "cdsctl admin organization delete <organization-name>",
	Args: []cli.Arg{
		{
			Name: "organization-name",
		},
	},
}

func adminOrganizationDeleteRun(v cli.Values) error {
	if err := client.AdminOrganizationDelete(context.Background(), v.GetString("organization-name")); err != nil {
		return err
	}
	return nil
}

var adminOrganizationUserMigrateCmd = cli.Command{
	Name:    "user-migrate",
	Short:   "Associate an organization to all empty without it",
	Example: "cdsctl admin organization user-migrate <organization-name>",
	Args: []cli.Arg{
		{
			Name: "organization-name",
		},
	},
}

func adminOrganizationUserMigrateRun(v cli.Values) error {
	if err := client.AdminOrganizationMigrateUser(context.Background(), v.GetString("organization-name")); err != nil {
		return err
	}
	return nil
}
