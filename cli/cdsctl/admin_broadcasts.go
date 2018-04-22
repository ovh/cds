package main

import (
	"reflect"

	"github.com/ovh/cds/cli"

	"github.com/spf13/cobra"
)

var (
	adminBroadcastsCmd = cli.Command{
		Name:  "broadcasts",
		Short: "Manage CDS broadcasts",
	}

	adminBroadcasts = cli.NewCommand(adminBroadcastsCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminBroadcastListCmd, adminBroadcastListRun, nil),
		})
)

var adminBroadcastListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS broadcasts",
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "level",
			ShortHand: "t",
			Usage:     "Filter broadcast by level",
			Default:   "",
		},
	},
}

func adminBroadcastListRun(v cli.Values) (cli.ListResult, error) {
	if v.GetString("level") == "" {
		srvs, err := client.Broadcasts()
		if err != nil {
			return nil, err
		}
		return cli.AsListResult(srvs), nil
	}

	srvs, err := client.BroadcastsByLevel(v.GetString("level"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(srvs), nil
}
