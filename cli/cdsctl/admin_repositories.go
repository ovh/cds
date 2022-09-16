package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminRepositoriesCmd = cli.Command{
	Name:  "repositories",
	Short: "Manage CDS repositories uService",
}

func adminRepositories() *cobra.Command {
	return cli.NewCommand(adminRepositoriesCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminRepositorisStatusCmd, adminRepositorisStatusRun, nil),
	})
}

func adminRepositorisStatusRun(_ cli.Values) (cli.ListResult, error) {
	services, err := client.ServicesByType(sdk.TypeRepositories)
	if err != nil {
		return nil, err
	}
	status := sdk.MonitoringStatus{}
	for _, srv := range services {
		status.Lines = append(status.Lines, srv.MonitoringStatus.Lines...)
	}
	return cli.AsListResult(status.Lines), nil
}

var adminRepositorisStatusCmd = cli.Command{
	Name:    "status",
	Short:   "display the status of repositories",
	Example: "cdsctl admin repositories status",
}
