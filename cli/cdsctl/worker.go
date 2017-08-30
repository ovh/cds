package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	workerCmd = cli.Command{
		Name:  "worker",
		Short: "Manage CDS worker",
	}

	worker = cli.NewCommand(workerCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(workerListCmd, workerListRun, nil),
			workerModel,
		})
)

var workerListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS workers",
}

func workerListRun(v cli.Values) (cli.ListResult, error) {
	workers, err := client.WorkerList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(workers), nil
}
