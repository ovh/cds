package main

import (
	"reflect"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	adminServicesCmd = cli.Command{
		Name:  "services",
		Short: "Manage CDS services",
	}

	adminServices = cli.NewCommand(adminServicesCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminServiceListCmd, adminServiceListRun, nil),
			cli.NewListCommand(adminServiceStatusCmd, adminServiceStatusRun, nil),
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
			Usage:     "Filter service by type: api, hatchery, hook, repository, vcs",
			Default:   "",
		},
	},
}

func adminServiceListRun(v cli.Values) (cli.ListResult, error) {
	srvs, err := client.ServicesByType(v.GetString("type"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(srvs), nil
}

var adminServiceStatusCmd = cli.Command{
	Name:  "status",
	Short: "Status CDS services",
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "type",
			ShortHand: "t",
			Usage:     "Filter service by type: api, hatchery, hook, repository, vcs",
			Default:   "",
		},
		{
			Kind:    reflect.String,
			Name:    "name",
			Usage:   "Filter service by name",
			Default: "",
		},
	},
}

func adminServiceStatusRun(v cli.Values) (cli.ListResult, error) {
	lines := []sdk.MonitoringStatusLine{}
	if v.GetString("name") != "" {
		srv, err := client.ServicesByName(v.GetString("name"))
		if err != nil {
			return nil, err
		}
		for _, l := range srv.MonitoringStatus.Lines {
			lines = append(lines, l)
		}
	} else if v.GetString("type") == "" {
		s, err := client.MonStatus()
		if err != nil {
			return nil, err
		}
		return cli.AsListResult(s.Lines), nil
	} else {
		srvs, err := client.ServicesByType(v.GetString("type"))
		if err != nil {
			return nil, err
		}
		for _, s := range srvs {
			for i := range s.MonitoringStatus.Lines {
				l := s.MonitoringStatus.Lines[i]
				lines = append(lines, l)
			}
		}
	}

	return cli.AsListResult(lines), nil
}
