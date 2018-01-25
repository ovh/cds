package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	dump "github.com/fsamin/go-dump"
	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
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

	var tags []string
	for _, tag := range run.Tags {
		tags = append(tags, fmt.Sprintf("%s:%s", tag.Tag, tag.Value))
	}

	type wtags struct {
		sdk.WorkflowRun
		Payload string `cli:"payload"`
		Tags    string `cli:"tags"`
	}

	var payload []string
	if v, ok := run.WorkflowNodeRuns[run.Workflow.RootID]; ok {
		if len(v) > 0 {
			e := dump.NewDefaultEncoder(new(bytes.Buffer))
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false
			pl, errm1 := e.ToStringMap(v[0].Payload)
			if errm1 != nil {
				return nil, errm1
			}
			for k, kv := range pl {
				payload = append(payload, fmt.Sprintf("%s:%s", k, kv))
			}
			payload = append(payload)
		}
	}

	wt := &wtags{*run, strings.Join(payload, " "), strings.Join(tags, " ")}
	return *wt, nil
}
