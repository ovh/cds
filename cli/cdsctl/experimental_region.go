package main

import (
	"context"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalRegionCmd = cli.Command{
	Name:    "region",
	Aliases: []string{"org", "orga"},
	Short:   "CDS Experimental region commands",
}

func experimentalRegion() *cobra.Command {
	return cli.NewCommand(experimentalRegionCmd, nil, []*cobra.Command{
		cli.NewCommand(regionAddCmd, regionAddFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(regionGetCmd, regionGetFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(regionListCmd, regionListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(regionDeleteCmd, regionDeleteFunc, nil, withAllCommandModifiers()...),
	})
}

var regionAddCmd = cli.Command{
	Name:    "add",
	Aliases: []string{"create"},
	Short:   "Create a new region",
	Example: "cdsctl region add <region_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "regionIdentifier"},
	},
}

func regionAddFunc(v cli.Values) error {
	reg := sdk.Region{Name: v.GetString("regionIdentifier")}
	if err := client.RegionAdd(context.Background(), reg); err != nil {
		return err
	}
	return nil
}

var regionGetCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "Get an region by its identifier",
	Example: "cdsctl region show <region_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "regionIdentifier"},
	},
}

func regionGetFunc(v cli.Values) (interface{}, error) {
	reg, err := client.RegionGet(context.Background(), v.GetString("regionIdentifier"))
	if err != nil {
		return reg, err
	}
	return reg, nil
}

var regionListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List all regions",
	Example: "cdsctl region list",
	Ctx:     []cli.Arg{},
}

func regionListFunc(_ cli.Values) (cli.ListResult, error) {
	regions, err := client.RegionList(context.Background())
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(regions), nil
}

var regionDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"remove", "rm"},
	Short:   "Remove region",
	Example: "cdsctl region delete",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{
			Name: "regionIdentifier",
		},
	},
}

func regionDeleteFunc(v cli.Values) error {
	err := client.RegionDelete(context.Background(), v.GetString("regionIdentifier"))
	if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil
	}
	return err
}
