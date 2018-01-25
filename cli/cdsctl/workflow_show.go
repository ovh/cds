package main

import (
	"github.com/ovh/cds/cli"
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
	w, err := client.WorkflowGet(v[_ProjectKey], v[_WorkflowName])
	if err != nil {
		return nil, err
	}
	return *w, nil
}
