package main

import (
	"context"
	"fmt"
	"strings"
	"time"

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
			cli.NewCommand(workerDisableCmd, workerDisableRun, nil),
			workerModel,
		})
)

var workerListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS workers",
}

func workerListRun(v cli.Values) (cli.ListResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	workers, err := client.WorkerList(ctx)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(workers), nil
}

var workerDisableCmd = cli.Command{
	Name:  "disable",
	Short: "Disable CDS workers",
	Long: `Disable one on more CDS worker by their names. 

For example if your want to disable all CDS workers you can run:
	
$ cdsctl worker disable $(cdsctl worker list)`,
	VariadicArgs: cli.Arg{
		Name: "name",
	},
}

func workerDisableRun(v cli.Values) error {
	names := v.GetStringSlice("name")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	workers, err := client.WorkerList(ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		var found bool
		for _, w := range workers {
			if w.ID == n || strings.ToLower(w.Name) == strings.ToLower(n) {
				found = true
				fmt.Printf("Disabling worker %s [status %s]... ", cli.Magenta(w.Name), w.Status)
				if err := client.WorkerDisable(context.Background(), w.ID); err != nil {
					fmt.Printf("Error disabling worker %s : %s\n", w.ID, err)
				} else {
					fmt.Printf("Done\n")
				}
			}
		}
		if !found {
			fmt.Printf("Worker %s not found\n", n)
		}
	}

	return nil
}
