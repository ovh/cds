package stats

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdStatsDBServerStatus = &cobra.Command{
	Use:   "dbServerStatus",
	Short: "DB Stats: tatcli stats dbServerStatus",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli stats dbServerStatus --help\n")
			cmd.Usage()
		} else {
			out, err := internal.Client().StatsDBServerStatus()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
