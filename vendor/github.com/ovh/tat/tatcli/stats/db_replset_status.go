package stats

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdStatsDBReplSetGetStatus = &cobra.Command{
	Use:   "dbReplSetGetStatus",
	Short: "DB Stats: tatcli stats dbReplSetGetStatus",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli stats dbReplSetGetStatus --help\n")
			cmd.Usage()
		} else {
			out, err := internal.Client().StatsDBReplSetGetStatus()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
