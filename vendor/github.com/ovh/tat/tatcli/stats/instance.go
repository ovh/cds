package stats

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdStatsInstance = &cobra.Command{
	Use:   "instance",
	Short: "Info about current instance of engine",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli stats instance --help\n")
			cmd.Usage()
		} else {
			out, err := internal.Client().StatsInstance()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
