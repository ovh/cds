package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	workflowAdvancedCmd = cli.Command{
		Name:  "advanced",
		Short: "Manage Workflow Advanced",
	}

	workflowAdvanced = cli.NewCommand(workflowAdvancedCmd, nil,
		[]*cobra.Command{
			cli.NewDeleteCommand(workflowDeleteCmd, workflowDeleteRun, nil, withAllCommandModifiers()...),
			workflowAdvancedRunNumber,
		})

	workflowAdvancedRunNumberCmd = cli.Command{
		Name:  "number",
		Short: "Manage Workflow Run Number",
	}

	workflowAdvancedRunNumber = cli.NewCommand(workflowAdvancedRunNumberCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(workflowRunNumberGetCmd, workflowRunNumberGetRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(workflowRunNumberSetCmd, workflowRunNumberSetRun, nil, withAllCommandModifiers()...),
		})
)

var workflowRunNumberGetCmd = cli.Command{
	Name:    "get",
	Short:   "Get a Workflow Run Number",
	Example: `cdsctl workflow advanced number set MYPROJECT my-workflow`,
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

func workflowRunNumberGetRun(v cli.Values) error {
	runNumber, err := client.WorkflowRunNumberGet(v[_ProjectKey], v[_WorkflowName])
	if err != nil {
		return err
	}
	fmt.Printf("%d\n", runNumber.Num)
	return nil
}

func workflowRunNumberSetRun(v cli.Values) error {
	number, err := strconv.ParseInt(v["number"], 10, 64)
	if err != nil {
		return fmt.Errorf("number parameter have to be an integer")
	}

	if err := client.WorkflowRunNumberSet(v[_ProjectKey], v[_WorkflowName], number); err != nil {
		return err
	}

	return nil
}
