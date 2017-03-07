package queue

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
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

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Prj", "App", "Status", "Job", "Queued", "Requirements"})
			table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
			table.SetCenterSeparator("|")

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
				table.Append([]string{prj, app, pb.Status, pb.Job.Action.Name, round(duration, time.Second).String(), req})
			}
			table.SetFooter([]string{"", "Total", fmt.Sprintf("%d", len(pbs)), "Max Queued", round(maxQueued, time.Second).String(), ""})

			table.Render()
		},
	}
)

func round(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}

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
