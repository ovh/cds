package main

import (
	"github.com/ovh/cds/cli"
	"github.com/spf13/cobra"
)

var adminWorkflowsCmd = cli.Command{
	Name:    "workflows",
	Aliases: []string{"workflow"},
	Short:   "Manage CDS workflows",
}

func adminWorkflows() *cobra.Command {
	return cli.NewCommand(adminWorkflowsCmd, nil, []*cobra.Command{
		cli.NewCommand(adminWorkflowUpdateMaxRunCmd, adminWorkflowUpdateMaxRun, nil),
	})
}

var adminWorkflowUpdateMaxRunCmd = cli.Command{
	Name:  "maxrun",
	Short: "Update the maximum number of workflow executions",
	Args: []cli.Arg{
		{
			Name: "projectKey",
		},
		{
			Name: "workflowName",
		},
		{
			Name: "maxRuns",
		},
	},
}

func adminWorkflowUpdateMaxRun(v cli.Values) error {
	maxRuns, err := v.GetInt64("maxRuns")
	if err != nil {
		return err
	}
	return client.AdminWorkflowUpdateMaxRuns(v.GetString("projectKey"), v.GetString("workflowName"), maxRuns)
}
