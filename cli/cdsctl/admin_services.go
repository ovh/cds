package main

import (
	"reflect"

	"github.com/ovh/cds/cli"

	"github.com/spf13/cobra"
)

var (
	adminServicesCmd = cli.Command{
		Name:  "services",
		Short: "Manage CDS services",
	}

	adminServices = cli.NewCommand(adminServicesCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminServiceListCmd, adminServiceListRun, nil),
		})
)

var adminServiceListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS services",
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "type",
			ShortHand: "t",
			Usage:     "Filter service by type",
			Default:   "",
		},
	},
}

func adminServiceListRun(v cli.Values) (cli.ListResult, error) {
	if v.GetString("type") == "" {
		srvs, err := client.Services()
		if err != nil {
			return nil, err
		}
		return cli.AsListResult(srvs), nil
	}

	srvs, err := client.ServicesByType(v.GetString("type"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(srvs), nil
}
