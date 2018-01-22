package stats

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdStatsDBReplSetGetConfig = &cobra.Command{
	Use:   "dbReplSetGetConfig",
	Short: "DB Stats: tatcli stats dbReplSetGetConfig",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli stats dbReplSetGetConfig --help\n")
			cmd.Usage()
		} else {
			out, err := internal.Client().StatsDBReplSetGetConfig()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
