package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalWorkerCmd = cli.Command{
	Name:    "worker",
	Aliases: []string{"workers"},
	Short:   "CDS Experimental worker commands",
}

func experimentalWorker() *cobra.Command {
	return cli.NewCommand(experimentalWorkerCmd, nil, []*cobra.Command{
		cli.NewListCommand(workerV2ListCmd, workerV2ListFunc, nil, withAllCommandModifiers()...),
	})
}

var workerV2ListCmd = cli.Command{
	Name:    "list",
	Example: "cdsctl experimental worker list",
}

func workerV2ListFunc(v cli.Values) (cli.ListResult, error) {
	workers, err := client.V2WorkerList(context.Background())
	if err != nil {
		return nil, err
	}

	return cli.AsListResult(workers), err
}
