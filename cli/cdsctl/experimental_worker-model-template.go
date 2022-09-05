package main

import (
	"context"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalWorkerModelTemplateCmd = cli.Command{
	Name:    "template",
	Aliases: []string{},
	Short:   "CDS Experimental worker-model template commands",
}

func experimentalWorkerModelTemplate() *cobra.Command {
	return cli.NewCommand(experimentalWorkerModelTemplateCmd, nil, []*cobra.Command{
		cli.NewListCommand(wmTemplateListCmd, wmTemplateListFunc, nil, withAllCommandModifiers()...),
	})
}

var wmTemplateListCmd = cli.Command{
	Name:    "list",
	Example: "cdsctl worker-model-template list",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository"},
	},
	Flags: []cli.Flag{
		{Name: "branch", Usage: "Filter on a specific branch"},
	},
}

func wmTemplateListFunc(v cli.Values) (cli.ListResult, error) {
	vcsName := v.GetString("vcs-name")
	repositoryName := v.GetString("repository")

	branch := v.GetString("branch")
	var filter *cdsclient.WorkerModelTemplateFilter
	if branch != "" {
		filter = &cdsclient.WorkerModelTemplateFilter{
			Branch: branch,
		}
	}

	tmpls, err := client.WorkerModelTemplateList(context.Background(), v.GetString(_ProjectKey), vcsName, repositoryName, filter)
	if err != nil {
		return nil, err
	}

	type Result struct {
		Name string `cli:"name"`
		Type string `cli:"type"`
	}
	results := make([]Result, 0, len(tmpls))
	for _, t := range tmpls {
		tmplType := "docker"
		if t.VM != nil {
			tmplType = "vm"
		}
		results = append(results, Result{Name: t.Name, Type: tmplType})
	}

	return cli.AsListResult(results), err
}
