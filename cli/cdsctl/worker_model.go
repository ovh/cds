package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	workerModelCmd = cli.Command{
		Name:  "model",
		Short: "Manage Worker Model",
	}

	workerModel = cli.NewCommand(workerModelCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(workerModelListCmd, workerModelListRun, nil),
		})
)

var workerModelListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS worker models",
}

func workerModelListRun(v cli.Values) (cli.ListResult, error) {
	workerModels, err := client.WorkerModels()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(workerModels), nil
}
