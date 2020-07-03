package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

var workflowLabelCmd = cli.Command{
	Name:    "label",
	Aliases: []string{"labels"},
	Short:   "Manage Workflow Label",
}

func workflowLabel() *cobra.Command {
	return cli.NewCommand(workflowLabelCmd, nil, []*cobra.Command{
		cli.NewListCommand(workflowLabelListCmd, workflowLabelListRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowLabelAddCmd, workflowLabelAddRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowLabelDeleteCmd, workflowLabelDeleteRun, nil, withAllCommandModifiers()...),
	})
}

var workflowLabelListCmd = cli.Command{
	Name:  "list",
	Short: "List labels of one workflow",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
}

func workflowLabelListRun(v cli.Values) (cli.ListResult, error) {
	wf, err := client.WorkflowGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName), cdsclient.WithLabels())
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(wf.Labels), nil
}

var workflowLabelAddCmd = cli.Command{
	Name:  "add",
	Short: "Add label on one workflow",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Args: []cli.Arg{
		{Name: "label"},
	},
}

func workflowLabelAddRun(v cli.Values) error {
	labelName := v.GetString("label")
	return client.WorkflowLabelAdd(v.GetString(_ProjectKey), v.GetString(_WorkflowName), labelName)
}

var workflowLabelDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"rm"},
	Short:   "Delete label from one workflow",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Args: []cli.Arg{
		{Name: "label"},
	},
}

func workflowLabelDeleteRun(v cli.Values) error {
	labelName := v.GetString("label")
	wf, err := client.WorkflowGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName), cdsclient.WithLabels())
	if err != nil {
		return err
	}

	var labelID int64
	for _, v := range wf.Labels {
		if v.Name == labelName {
			labelID = v.ID
			break
		}
	}

	return client.WorkflowLabelDelete(v.GetString(_ProjectKey), v.GetString(_WorkflowName), labelID)
}
