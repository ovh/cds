package stats

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdStatsDBStats = &cobra.Command{
	Use:   "dbstats",
	Short: "DB Stats: tatcli stats dbstats",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli stats db --help\n")
		} else {
			out, err := internal.Client().StatsDBStats()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
