package main

import (
	"context"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalWorkerModelCmd = cli.Command{
	Name:    "worker-model",
	Aliases: []string{"wm"},
	Short:   "CDS Experimental worker model commands",
}

func experimentalWorkerModel() *cobra.Command {
	return cli.NewCommand(experimentalWorkerModelCmd, nil, []*cobra.Command{
		cli.NewListCommand(wmListCmd, workerModelListFunc, nil, withAllCommandModifiers()...),
		experimentalWorkerModelTemplate(),
	})
}

var wmListCmd = cli.Command{
	Name:    "list",
	Example: "cdsctl worker-model list",
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

func workerModelListFunc(v cli.Values) (cli.ListResult, error) {
	vcsName := v.GetString("vcs-name")
	repositoryName := v.GetString("repository")

	branch := v.GetString("branch")
	var filter *cdsclient.WorkerModelV2Filter
	if branch != "" {
		filter = &cdsclient.WorkerModelV2Filter{
			Branch: branch,
		}
	}

	wms, err := client.WorkerModelv2List(context.Background(), v.GetString(_ProjectKey), vcsName, repositoryName, filter)
	if err != nil {
		return nil, err
	}

	type Result struct {
		Name string `cli:"name"`
		Type string `cli:"type"`
	}
	results := make([]Result, 0, len(wms))
	for _, t := range wms {
		var modelType string
		switch {
		case t.Docker != nil:
			modelType = "docker"
		case t.VSphere != nil:
			modelType = "vsphere"
		case t.Openstack != nil:
			modelType = "openstack"
		default:
			modelType = "unknown"
		}
		results = append(results, Result{Name: t.Name, Type: modelType})
	}
	return cli.AsListResult(results), err
}
