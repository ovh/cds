package main

import (
	"bytes"
	"fmt"
	"strings"

	dump "github.com/fsamin/go-dump"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var workflowShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS workflow",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
}

func workflowShowRun(v cli.Values) (interface{}, error) {
	var runNumber int64
	if v.GetString("run-number") != "" {
		var errl error
		runNumber, errl = v.GetInt64("run-number")
		if errl != nil {
			return nil, errl
		}
	}

	if runNumber == 0 {
		w, err := client.WorkflowGet(v[_ProjectKey], v[_WorkflowName])
		if err != nil {
			return nil, err
		}
		return *w, nil
	}

	w, err := client.WorkflowRunGet(v[_ProjectKey], v[_WorkflowName], runNumber)
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, tag := range w.Tags {
		tags = append(tags, fmt.Sprintf("%s:%s", tag.Tag, tag.Value))
	}

	type wtags struct {
		sdk.WorkflowRun
		Payload string `cli:"payload"`
		Tags    string `cli:"tags"`
	}

	var payload []string
	if v, ok := w.WorkflowNodeRuns[w.Workflow.RootID]; ok {
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

	wt := &wtags{*w, strings.Join(payload, " "), strings.Join(tags, " ")}
	return *wt, nil
}
