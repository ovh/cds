package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminServicesCmd = cli.Command{
	Name:  "services",
	Short: "Manage CDS services",
}

func adminServices() *cobra.Command {
	return cli.NewCommand(adminServicesCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminServiceListCmd, adminServiceListRun, nil),
		cli.NewListCommand(adminServiceStatusCmd, adminServiceStatusRun, nil),
		cli.NewCommand(adminServiceGetCmd, adminServiceGetRun, nil),
		cli.NewDeleteCommand(adminServiceDeleteCmd, adminServiceDeleteRun, nil, withAllCommandModifiers()...),
	})
}

var adminServiceListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS services",
	Flags: []cli.Flag{
		{
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
			Name:      "type",
			ShortHand: "t",
			Usage:     "Filter service by type: api, hatchery, hook, repository, vcs",
			Default:   "",
		},
		{
			Name:    "name",
			Usage:   "Filter service by name",
			Default: "",
		},
	},
}

var adminServiceGetCmd = cli.Command{
	Name:  "request",
	Short: "request GET on a CDS service",
	Example: `
## How to get the goroutine of the service named hatcheryLocal:
` + "```bash" + `
cdsctl admin services request --name hatcheryLocal --query /debug/pprof/goroutine\?debug\=2
` + "```" + `

`,
	Flags: []cli.Flag{
		{
			Name:    "name",
			Usage:   "service name",
			Default: "",
		},
		{
			Name:    "query",
			Usage:   "http query, example: '/debug/pprof/goroutine?debug=2'",
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

func adminServiceGetRun(v cli.Values) error {
	btes, err := client.ServiceNameCallGET(v.GetString("name"), v.GetString("query"))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}

var adminServiceDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS service from registered service",
	VariadicArgs: cli.Arg{
		Name: "name",
	},
}

func adminServiceDeleteRun(v cli.Values) error {
	if v.GetString("name") == "" {
		return fmt.Errorf("name for service is mandatory")
	}
	for _, n := range strings.Split(v.GetString("name"), ",") {
		if err := client.ServiceDelete(n); err != nil {
			return err
		}
	}
	return nil
}
