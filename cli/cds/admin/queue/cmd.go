package queue

import (
	"fmt"
	"time"

	"github.com/fatih/color"
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
				sdk.Exit("You are not allowed to run this command\n")
			}

			pbs, err := sdk.GetBuildQueue()
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			if len(pbs) == 0 {
				sdk.Exit("No job in queue\n")
			}

			red := color.New(color.BgRed).Add(color.FgWhite)

			var maxQueued time.Duration
			for _, pb := range pbs {
				req := ""
				for _, r := range pb.Job.Action.Requirements {
					req += fmt.Sprintf("%s(%s):%s ", r.Name, r.Type, r.Value)
				}
				prj := getVarsInPbj("cds.project", pb.Parameters)
				app := getVarsInPbj("cds.application", pb.Parameters)
				duration := time.Since(pb.Queued)
				if maxQueued < duration {
					maxQueued = duration
				}

				if duration > 10*time.Second {
					red.Printf(sdk.Round(duration, time.Second).String())
				} else {
					fmt.Printf(sdk.Round(duration, time.Second).String())
				}

				if pb.BookedBy.ID != 0 {
					fmt.Printf(" BOOKED(%d) ", pb.BookedBy.ID)
				} else {
					fmt.Printf("\t")
				}

				fmt.Printf(" \t%s➤%s➤%s \t%s\n", prj, app, pb.Job.Action.Name, req)
			}
			fmt.Printf("max:%s\n", sdk.Round(maxQueued, time.Second).String())
		},
	}
)

func getVarsInPbj(key string, ps []sdk.Parameter) string {
	for _, p := range ps {
		if p.Name == key {
			return p.Value
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(queueStatusCmd)
}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
