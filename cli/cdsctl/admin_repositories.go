package main

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	adminRepositoriesCmd = cli.Command{
		Name:  "repositories",
		Short: "Manage CDS Repositories service",
	}

	adminRepositories = cli.NewCommand(adminRepositoriesCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminRepositoriesStatusCmd, adminRepositoriesStatusRun, nil),
		})
)

var adminRepositoriesStatusCmd = cli.Command{
	Name:  "status",
	Short: "Show CDS Repositories Status",
}

func adminRepositoriesStatusRun(v cli.Values) (cli.ListResult, error) {
	btes, err := client.ServiceCallGET("repositories", "/mon/status")
	if err != nil {
		return nil, err
	}
	ts := sdk.MonitoringStatus{}
	if err := json.Unmarshal(btes, &ts); err != nil {
		return nil, err
	}
	return cli.AsListResult(ts.Lines), nil
}
