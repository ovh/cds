package main

import (
	"github.com/ovh/cds/cli"
	"github.com/spf13/cobra"
)

var (
	workflowCmd = cli.Command{
		Name:  "workflow",
		Short: "Manage CDS workflow",
	}

	workflow = cli.NewCommand(workflowCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(workflowListCmd, workflowListRun, nil),
			cli.NewGetCommand(workflowShowCmd, workflowShowRun, nil),
		})
)

var workflowListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS workflows",
	Args: []cli.Arg{
		{Name: "project-key"},
	},
}

func workflowListRun(v cli.Values) (cli.ListResult, error) {
	w, err := client.WorkflowList(v["project-key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(w), nil
}

var workflowShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "name"},
	},
}

func workflowShowRun(v cli.Values) (interface{}, error) {
	w, err := client.WorkflowGet(v["project-key"], v["name"])
	if err != nil {
		return nil, err
	}
	return *w, nil
}
