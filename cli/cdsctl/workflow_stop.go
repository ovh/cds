package main

import (
	"fmt"
	"strconv"

	"github.com/ovh/cds/cli"
)

var workflowStopCmd = cli.Command{
	Name:  "stop",
	Short: "Stop a CDS workflow or a specific node name",
	Long:  "Stop a CDS workflow or a specific node name",
	Example: `
		cdsctl workflow stop # Stop the workflow run for the current repo and the current hash
		cdsctl workflow stop MYPROJECT myworkflow 5 # To stop a workflow run on number 5
		cdsctl workflow stop MYPROJECT myworkflow 5 compile # To stop a workflow node run on workflow run 5
	`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	OptionalArgs: []cli.Arg{
		{Name: "run-number"},
		{Name: "node-name"},
	},
}

func workflowStopRun(v cli.Values) error {

	var runNumber int64
	var errRunNumber error
	// If no run number, get the latest
	runNumberStr := v.GetString("run-number")
	if runNumberStr != "" {
		runNumber, errRunNumber = strconv.ParseInt(runNumberStr, 10, 64)
	} else {
		runNumber, errRunNumber = workflowNodeForCurrentRepo(v[_ProjectKey], v.GetString(_WorkflowName))
	}
	if errRunNumber != nil {
		return errRunNumber
	}

	var fromNodeID int64
	if v.GetString("node-name") != "" {
		if runNumber <= 0 {
			return fmt.Errorf("You can use flag node-name without flag run-number")
		}
		wr, err := client.WorkflowRunGet(v[_ProjectKey], v[_WorkflowName], runNumber)
		if err != nil {
			return err
		}
		for _, wnrs := range wr.WorkflowNodeRuns {
			if wnrs[0].WorkflowNodeName == v.GetString("node-name") {
				fromNodeID = wnrs[0].ID
				break
			}
		}
		if fromNodeID == 0 {
			return fmt.Errorf("Node not found")
		}
	}

	if fromNodeID != 0 {
		wNodeRun, err := client.WorkflowNodeStop(v[_ProjectKey], v[_WorkflowName], runNumber, fromNodeID)
		if err != nil {
			return err
		}
		fmt.Printf("Workflow node %s from workflow %s #%d has been stopped\n", v.GetString("node-name"), v[_WorkflowName], wNodeRun.Number)
	} else {
		w, err := client.WorkflowStop(v[_ProjectKey], v[_WorkflowName], runNumber)
		if err != nil {
			return err
		}
		fmt.Printf("Workflow %s #%d has been stopped\n", v[_WorkflowName], w.Number)
	}

	return nil
}
