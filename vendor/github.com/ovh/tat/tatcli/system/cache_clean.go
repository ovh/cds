package system

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdSystemCacheClean = &cobra.Command{
	Use:   "cacheclean",
	Short: "Clean Cache: tatcli system cacheclean",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli system cacheclean --help\n")
		} else {
			out, err := internal.Client().SystemCacheClean()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
