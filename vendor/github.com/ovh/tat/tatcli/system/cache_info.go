package system

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdSystemCacheInfo = &cobra.Command{
	Use:   "cacheinfo",
	Short: "Info on Cache: tatcli system cacheinfo",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			internal.Exit("Invalid argument: tatcli system cacheinfo --help\n")
		} else {
			out, err := internal.Client().SystemCacheInfo()
			internal.Check(err)
			internal.Print(out)
		}
	},
}
