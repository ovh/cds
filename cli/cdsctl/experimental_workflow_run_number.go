package main

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalWorkflowRunNumberCmd = cli.Command{
	Name:  "run-number",
	Short: "CDS Experimental workflow run number commands",
}

func experimentalWorkflowRunNumber() *cobra.Command {
	return cli.NewCommand(experimentalWorkflowRunNumberCmd, nil, []*cobra.Command{
		cli.NewGetCommand(workflowV2RunNumberShowCmd, workflowV2RunNumberShowFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowV2RunNumberSetCmd, workflowV2RunNumberSetFunc, nil, withAllCommandModifiers()...),
	})
}

var workflowV2RunNumberShowCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "Show the current run number of the workflow, 0 if it never ran",
	Example: "cdsctl experimental workflow run-number show <project_key> <vcs_identifier> <repository_identifier> <workflow_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repository_identifier"},
		{Name: "workflow_name"},
	},
	Mcp: true,
}

func workflowV2RunNumberShowFunc(v cli.Values) (interface{}, error) {
	return client.WorkflowV2RunNumberGet(context.Background(), v.GetString("proj_key"), v.GetString("vcs_identifier"), v.GetString("repository_identifier"), v.GetString("workflow_name"))
}

var workflowV2RunNumberSetCmd = cli.Command{
	Name:  "set",
	Short: "Set the current run number of the workflow: its next run gets this number plus one",
	Long: `Set the current run number of the workflow: its next run gets this number plus one.

The number cannot be lower than the highest run number of the existing runs of the workflow.`,
	Example: "cdsctl experimental workflow run-number set <project_key> <vcs_identifier> <repository_identifier> <workflow_name> <run_number>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repository_identifier"},
		{Name: "workflow_name"},
		{Name: "run_number"},
	},
}

func workflowV2RunNumberSetFunc(v cli.Values) error {
	runNumber, err := strconv.ParseInt(v.GetString("run_number"), 10, 64)
	if err != nil || runNumber < 0 {
		return cli.NewError("run_number must be a positive integer")
	}
	return client.WorkflowV2RunNumberSet(context.Background(), v.GetString("proj_key"), v.GetString("vcs_identifier"), v.GetString("repository_identifier"), v.GetString("workflow_name"), runNumber)
}
