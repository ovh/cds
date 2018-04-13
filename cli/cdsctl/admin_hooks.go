package main

import (
	"encoding/json"
	"reflect"

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
		})
)

var adminHooksTaskListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS Hooks Tasks",
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "name",
			ShortHand: "t",
			Usage:     "get task from a hook service with this name. Optional if you have only one hooks service",
			Default:   "",
		},
	},
}

func adminHooksTaskListRun(v cli.Values) (cli.ListResult, error) {

	if v.GetString("name") == "" {
		srvs, err := client.Services()
		if err != nil {
			return nil, err
		}
		return cli.AsListResult(srvs), nil
	}

	btes, err := client.ServiceCallGET("hooks", "", "/task")
	if err != nil {
		return nil, err
	}
	ts := []sdk.Task{}
	if err := json.Unmarshal(btes, &ts); err != nil {
		return nil, err
	}
	return cli.AsListResult(ts), nil
}
