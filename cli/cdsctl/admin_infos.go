package main

import (
	"reflect"

	"github.com/ovh/cds/cli"

	"github.com/spf13/cobra"
)

var (
	adminInfosCmd = cli.Command{
		Name:  "infos",
		Short: "Manage CDS infos",
	}

	adminInfos = cli.NewCommand(adminInfosCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminInfoListCmd, adminInfoListRun, nil),
		})
)

var adminInfoListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS infos",
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "level",
			ShortHand: "t",
			Usage:     "Filter info by level",
			Default:   "",
		},
	},
}

func adminInfoListRun(v cli.Values) (cli.ListResult, error) {
	if v.GetString("level") == "" {
		srvs, err := client.Infos()
		if err != nil {
			return nil, err
		}
		return cli.AsListResult(srvs), nil
	}

	srvs, err := client.InfosByLevel(v.GetString("level"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(srvs), nil
}
