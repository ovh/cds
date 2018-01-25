package stats

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdStatsDBCollections = &cobra.Command{
	Use:   "dbCollections",
	Short: "DB Stats on each collection: tatcli stats dbCollections",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli stats dbCollections --help\n")
			cmd.Usage()
		} else {
			out, err := internal.Client().StatsDBCollections()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
