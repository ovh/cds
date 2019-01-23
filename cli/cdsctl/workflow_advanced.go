package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var workflowAdvancedCmd = cli.Command{
	Name:  "advanced",
	Short: "Manage Workflow Advanced",
}

func workflowAdvanced() *cobra.Command {
	return cli.NewCommand(workflowAdvancedCmd, nil, []*cobra.Command{
		cli.NewDeleteCommand(workflowDeleteCmd, workflowDeleteRun, nil, withAllCommandModifiers()...),
		workflowAdvancedRunNumber(),
	})
}

var workflowAdvancedRunNumberCmd = cli.Command{
	Name:  "number",
	Short: "Manage Workflow Run Number",
}

func workflowAdvancedRunNumber() *cobra.Command {
	return cli.NewCommand(workflowAdvancedRunNumberCmd, nil, []*cobra.Command{
		cli.NewGetCommand(workflowRunNumberShowCmd, workflowRunNumberShowRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowRunNumberSetCmd, workflowRunNumberSetRun, nil, withAllCommandModifiers()...),
	})
}

var workflowRunNumberShowCmd = cli.Command{
	Name:    "show",
	Short:   "Show a Workflow Run Number",
	Example: `cdsctl workflow advanced number show MYPROJECT my-workflow`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
}

var workflowRunNumberSetCmd = cli.Command{
	Name:    "set",
	Short:   "Set a Workflow Run Number",
	Example: `cdsctl workflow advanced number set MYPROJECT my-workflow 22`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Args: []cli.Arg{
		{Name: "number"},
	},
}

func workflowRunNumberShowRun(v cli.Values) (interface{}, error) {
	return client.WorkflowRunNumberGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName))
}

func workflowRunNumberSetRun(v cli.Values) error {
	number, err := strconv.ParseInt(v.GetString("number"), 10, 64)
	if err != nil {
		return fmt.Errorf("number parameter have to be an integer")
	}

	return client.WorkflowRunNumberSet(v.GetString(_ProjectKey), v.GetString(_WorkflowName), number)
}
