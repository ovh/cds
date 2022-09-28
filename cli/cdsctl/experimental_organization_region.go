package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalOrganizationRegionCmd = cli.Command{
	Name:    "region",
	Aliases: []string{"region"},
	Short:   "CDS Experimental organization region commands",
}

func experimentalOrganizationRegion() *cobra.Command {
	return cli.NewCommand(experimentalOrganizationRegionCmd, nil, []*cobra.Command{
		cli.NewCommand(organizationRegionAddCmd, organizationRegionAddFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(organizationRegionListCmd, organizationRegionListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(organizationRegionDeleteCmd, organizationRegionDeleteFunc, nil, withAllCommandModifiers()...),
	})
}

var organizationRegionAddCmd = cli.Command{
	Name:    "add",
	Short:   "Add a region in an organization",
	Example: "cdsctl organization region add <organization_name> <region_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "organizationIdentifier"},
		{Name: "regionIdentifier"},
	},
}

func organizationRegionAddFunc(v cli.Values) error {
	reg, err := client.RegionGet(context.Background(), v.GetString("regionIdentifier"))
	if err != nil {
		return err
	}
	if err := client.OrganizationRegionAllow(context.Background(), v.GetString("organizationIdentifier"), reg); err != nil {
		return err
	}
	return nil
}

var organizationRegionListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List region allowed in a organization",
	Example: "cdsctl organization region list <organization_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "organizationIdentifier"},
	},
}

func organizationRegionListFunc(v cli.Values) (cli.ListResult, error) {
	regs, err := client.OrganizationRegionList(context.Background(), v.GetString("organizationIdentifier"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(regs), nil
}

var organizationRegionDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete a region from an organization",
	Example: "cdsctl organization region delete <organization_name> <region_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "organizationIdentifier"},
		{Name: "regionIdentifier"},
	},
}

func organizationRegionDeleteFunc(v cli.Values) error {
	if err := client.OrganizationRegionRemove(context.Background(), v.GetString("organizationIdentifier"), v.GetString("regionIdentifier")); err != nil {
		return err
	}
	return nil
}
