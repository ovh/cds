package worker

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cli/cds/worker/model"
)

func init() {
	Cmd.AddCommand(model.Cmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(listCmd)
}

// Cmd worker
var Cmd = &cobra.Command{
	Use:     "worker",
	Short:   "Get information about workers status",
	Long:    ``,
	Aliases: []string{"w"},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "cds worker status",
	Run:   workerStatus,
}

func workerStatus(cmd *cobra.Command, args []string) {

	ms, err := sdk.GetWorkerModelStatus()
	if err != nil {
		sdk.Exit("Error: Cannot get model status (%s)\n", err)
	}

	for i := range ms {
		var warning string
		if ms[i].CurrentCount < ms[i].WantedCount || ms[i].CurrentCount > ms[i].WantedCount+5 {
			warning = "/!\\"
		}
		fmt.Printf("- %-10s ( %-2d / %2d ) %s\n", ms[i].ModelName, ms[i].CurrentCount, ms[i].WantedCount, warning)
	}
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "cds worker list",
	Run:   workerList,
}

func workerList(cmd *cobra.Command, args []string) {

	workers, err := sdk.GetWorkers()
	if err != nil {
		sdk.Exit("Error: Cannot get worker (%s)\n", err)
	}

	for _, w := range workers {
		fmt.Printf("- %-30s %s\n", w.Name, w.Status)
	}
}
