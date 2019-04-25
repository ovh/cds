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
	w, err := client.WorkflowGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName))
	if err != nil {
		return nil, err
	}
	wrkflw := struct {
		ProjectKey  string `cli:"project_key"`
		Name        string `cli:"name"`
		Description string `cli:"description"`
		From        string `cli:"from"`
		URL         string `cli:"url"`
		API         string `cli:"api"`
	}{
		ProjectKey:  w.ProjectKey,
		Name:        w.Name,
		Description: w.Description,
		From:        w.FromRepository,
		URL:         w.URLs.UIURL,
		API:         w.URLs.APIURL,
	}
	return wrkflw, nil
}
