package main

import "github.com/ovh/cds/cli"

var workflowListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS workflows",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func workflowListRun(v cli.Values) (cli.ListResult, error) {
	w, err := client.WorkflowList(v[_ProjectKey])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(w), nil
}
