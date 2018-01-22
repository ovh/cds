package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserAddFavoriteTag = &cobra.Command{
	Use:   "addFavoriteTag",
	Short: "Add a favorite Tag: tatcli user addFavoriteTag <tag>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserAddFavoriteTag(args[0])
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user addFavoriteTag --help\n")
		}
	},
}
