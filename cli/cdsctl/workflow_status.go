package main

import (
	"strconv"

	"github.com/ovh/cds/cli"
)

var workflowStatusCmd = cli.Command{
	Name:  "status",
	Short: "Check the status of the run",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	OptionalArgs: []cli.Arg{
		{Name: "run-number"},
	},
}

func workflowStatusRun(v cli.Values) (interface{}, error) {
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
		return nil, errRunNumber
	}

	run, err := client.WorkflowRunGet(v[_ProjectKey], v.GetString(_WorkflowName), runNumber)
	if err != nil {
		return nil, err
	}
	return run, nil
}
