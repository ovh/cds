package main

import (
	"context"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalHatcheryCmd = cli.Command{
	Name:  "hatchery",
	Short: "CDS Experimental hatchery commands",
}

func experimentalHatchery() *cobra.Command {
	return cli.NewCommand(experimentalHatcheryCmd, nil, []*cobra.Command{
		cli.NewGetCommand(hatcheryAddCmd, hatcheryAddFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(hatcheryGetCmd, hatcheryGetFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(hatcheryListCmd, hatcheryListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(hatcheryDeleteCmd, hatcheryDeleteFunc, nil, withAllCommandModifiers()...),
	})
}

var hatcheryAddCmd = cli.Command{
	Name:    "add",
	Aliases: []string{"create"},
	Short:   "Create a new hatchery",
	Example: "cdsctl hatchery add <hatchery_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "hatcheryIdentifier"},
	},
}

func hatcheryAddFunc(v cli.Values) (interface{}, error) {
	h := sdk.Hatchery{Name: v.GetString("hatcheryIdentifier")}
	if err := client.HatcheryAdd(context.Background(), &h); err != nil {
		return nil, err
	}
	return h, nil
}

var hatcheryGetCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "Get an hatchery by its identifier",
	Example: "cdsctl hatchery show <hatchery_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "hatcheryIdentifier"},
	},
}

func hatcheryGetFunc(v cli.Values) (interface{}, error) {
	h, err := client.HatcheryGet(context.Background(), v.GetString("hatcheryIdentifier"))
	if err != nil {
		return h, err
	}
	return h, nil
}

var hatcheryListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List all hatcheries",
	Example: "cdsctl hatchery list",
	Ctx:     []cli.Arg{},
}

func hatcheryListFunc(_ cli.Values) (cli.ListResult, error) {
	hatcheries, err := client.HatcheryList(context.Background())
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(hatcheries), nil
}

var hatcheryDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"remove", "rm"},
	Short:   "Remove hatchery",
	Example: "cdsctl hatchery delete",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{
			Name: "hatcheryIdentifier",
		},
	},
}

func hatcheryDeleteFunc(v cli.Values) error {
	err := client.HatcheryDelete(context.Background(), v.GetString("hatcheryIdentifier"))
	if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil
	}
	return err
}
