package queue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var (
	rootCmd = &cobra.Command{
		Use:   "queue",
		Short: "CDS Admin Queue (admin only)",
	}

	queueStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "cds admin queue status",
		Run: func(cmd *cobra.Command, args []string) {
			if ok, err := sdk.IsAdmin(); !ok {
				if err != nil {
					fmt.Printf("Error : %v\n", err)
				}
				sdk.Exit("You are not allowed to run this command")
			}

			pbs, err := sdk.GetBuildQueue()
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			for _, pb := range pbs {
				fmt.Printf("PB: action:%s status:%s model:%s queued:%s Requirements:%+v\n", pb.Job.Action.Name, pb.Status, pb.Model, pb.Queued, pb.Job.Action.Requirements)
			}

		},
	}
)

func init() {
	rootCmd.AddCommand(queueStatusCmd)
}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
