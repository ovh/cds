package main

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	adminVCSCmd = cli.Command{
		Name:  "vcs",
		Short: "Manage CDS VCS service",
	}

	adminVCS = cli.NewCommand(adminVCSCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminVCSStatusCmd, adminVCSStatusRun, nil),
		})
)

var adminVCSStatusCmd = cli.Command{
	Name:  "status",
	Short: "Show CDS VCS Status",
}

func adminVCSStatusRun(v cli.Values) (cli.ListResult, error) {
	btes, err := client.ServiceCallGET("vcs", "/mon/status")
	if err != nil {
		return nil, err
	}
	ts := sdk.MonitoringStatus{}
	if err := json.Unmarshal(btes, &ts); err != nil {
		return nil, err
	}
	return cli.AsListResult(ts.Lines), nil
}
