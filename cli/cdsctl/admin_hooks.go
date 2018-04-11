package main

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	adminHooksCmd = cli.Command{
		Name:  "hooks",
		Short: "Manage CDS Hooks tasks",
	}

	adminHooks = cli.NewCommand(adminHooksCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminHooksTaskListCmd, adminHooksTaskListRun, nil),
			cli.NewListCommand(adminHooksStatusCmd, adminHooksStatusRun, nil),
		})
)

var adminHooksTaskListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS Hooks Tasks",
}

func adminHooksTaskListRun(v cli.Values) (cli.ListResult, error) {
	btes, err := client.ServiceCallGET("hooks", "/task")
	if err != nil {
		return nil, err
	}
	ts := []sdk.Task{}
	if err := json.Unmarshal(btes, &ts); err != nil {
		return nil, err
	}
	return cli.AsListResult(ts), nil
}

var adminHooksStatusCmd = cli.Command{
	Name:  "status",
	Short: "Show CDS VCS Status",
}

func adminHooksStatusRun(v cli.Values) (cli.ListResult, error) {
	btes, err := client.ServiceCallGET("hooks", "/mon/status")
	if err != nil {
		return nil, err
	}
	ts := sdk.MonitoringStatus{}
	if err := json.Unmarshal(btes, &ts); err != nil {
		return nil, err
	}
	return cli.AsListResult(ts.Lines), nil
}
