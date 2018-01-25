package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserAddFavoriteTopic = &cobra.Command{
	Use:   "addFavoriteTopic",
	Short: "Add a favorite Topic: tatcli user addFavoriteTopic <topicName>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserAddFavoriteTopic(args[0])
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user addFavoriteTopic --help\n")
		}
	},
}
