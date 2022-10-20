package main

import (
	"context"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalOrganizationCmd = cli.Command{
	Name:    "organization",
	Aliases: []string{"org", "orga"},
	Short:   "CDS Experimental organization commands",
}

func experimentalOrganization() *cobra.Command {
	return cli.NewCommand(experimentalOrganizationCmd, nil, []*cobra.Command{
		cli.NewCommand(organizationAddCmd, organizationAddFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(organizationGetCmd, organizationGetFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(organizationListCmd, organizationListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(organizationDeleteCmd, organizationDeleteFunc, nil, withAllCommandModifiers()...),
	})
}

var organizationAddCmd = cli.Command{
	Name:    "add",
	Aliases: []string{"create"},
	Short:   "Create a new organization",
	Example: "cdsctl organization add <organization_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "organizationIdentifier"},
	},
}

func organizationAddFunc(v cli.Values) error {
	orga := sdk.Organization{Name: v.GetString("organizationIdentifier")}
	if err := client.OrganizationAdd(context.Background(), orga); err != nil {
		return err
	}
	return nil
}

var organizationGetCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "Get an organization by its identifier",
	Example: "cdsctl organization show <organization_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "organizationIdentifier"},
	},
}

func organizationGetFunc(v cli.Values) (interface{}, error) {
	orga, err := client.OrganizationGet(context.Background(), v.GetString("organizationIdentifier"))
	if err != nil {
		return orga, err
	}
	return orga, nil
}

var organizationListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List all organizations",
	Example: "cdsctl organization list",
	Ctx:     []cli.Arg{},
}

func organizationListFunc(_ cli.Values) (cli.ListResult, error) {
	orgas, err := client.OrganizationList(context.Background())
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(orgas), nil
}

var organizationDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"remove", "rm"},
	Short:   "Remove organization",
	Example: "cdsctl organization delete",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{
			Name: "organizationIdentifier",
		},
	},
}

func organizationDeleteFunc(v cli.Values) error {
	err := client.OrganizationDelete(context.Background(), v.GetString("organizationIdentifier"))
	if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil
	}
	return err
}
