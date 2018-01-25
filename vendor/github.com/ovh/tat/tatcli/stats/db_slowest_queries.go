package stats

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdStatsDBSlowestQueries = &cobra.Command{
	Use:   "dbSlowestQueries",
	Short: "DB Stats slowest Queries: tatcli stats dbSlowestQueries",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli stats dbSlowestQueriess --help\n")
			cmd.Usage()
		} else {
			out, err := internal.Client().StatsDBSlowestQueries()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
