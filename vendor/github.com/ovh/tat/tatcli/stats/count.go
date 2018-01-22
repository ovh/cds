package stats

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdStatsCount = &cobra.Command{
	Use:   "count",
	Short: "Count all messages, groups, presences, users, groups, topics: tatcli stats count",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli stats count --help\n")
			cmd.Usage()
		} else {
			out, err := internal.Client().StatsCount()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
