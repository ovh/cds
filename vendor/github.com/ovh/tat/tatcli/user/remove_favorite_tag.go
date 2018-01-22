package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserRemoveFavoriteTag = &cobra.Command{
	Use:   "removeFavoriteTag",
	Short: "Remove a favorite Tag: tatcli user removeFavoriteTag <tag>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserRemoveFavoriteTag(args[0])
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user removeFavoriteTag --help\n")
		}
	},
}
