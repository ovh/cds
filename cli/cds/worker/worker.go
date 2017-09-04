package worker

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli/cds/worker/model"
	"github.com/ovh/cds/sdk"
)

func init() {
	Cmd.AddCommand(model.Cmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(killCmd)
}

// Cmd worker
var Cmd = &cobra.Command{
	Use:     "worker",
	Short:   "Get information about workers status",
	Long:    ``,
	Aliases: []string{"w"},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "cds worker list",
	Run:   workerList,
}

func workerList(cmd *cobra.Command, args []string) {

	models, err := sdk.GetWorkerModels()
	if err != nil {
		sdk.Exit("Error: cannot get worker models (%s)\n", err)
	}

	mmodels := make(map[int64]sdk.Model, len(models))
	for _, m := range models {
		mmodels[m.ID] = m
	}
	workers, err := sdk.GetWorkers()
	if err != nil {
		sdk.Exit("Error: Cannot get worker (%s)\n", err)
	}

	for _, w := range workers {
		m := "N/A"
		if _, ok := mmodels[w.ModelID]; ok {
			m = mmodels[w.ModelID].Type
		}
		fmt.Printf("%-10s %-50s %s\n", m, w.Name, w.Status)
	}
}

var killCmd = &cobra.Command{
	Use:   "disable",
	Short: "cds worker disable <name or id> [<name or id>]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}

		workers, err := sdk.GetWorkers()
		if err != nil {
			sdk.Exit("Error: Cannot get worker (%s)\n", err)
		}

		var exitCode int
		for _, id := range args {
			var found bool
			for _, w := range workers {
				if w.ID == id || strings.ToLower(w.Name) == strings.ToLower(id) {
					found = true
					fmt.Printf(" - Disabling worker %s [status %s]... ", w.Name, w.Status)
					if err := sdk.DisableWorker(w.ID); err != nil {
						fmt.Printf("Error disabling worker %s : %s\n", w.ID, err)
						exitCode++
					} else {
						fmt.Printf("Done\n")
					}
				}
			}
			if !found {
				fmt.Printf(" - Worker %s not found\n", id)
				exitCode++
			}
		}
		os.Exit(exitCode)
	},
}
