package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

var experimentalWorkerModelCmd = cli.Command{
	Name:    "worker-model",
	Aliases: []string{"wm"},
	Short:   "CDS Experimental worker model commands",
}

func experimentalWorkerModel() *cobra.Command {
	return cli.NewCommand(experimentalWorkerModelCmd, nil, []*cobra.Command{
		cli.NewListCommand(wmListCmd, workerModelListFunc, nil, withAllCommandModifiers()...),
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

	return cli.AsListResult(wms), err
}
