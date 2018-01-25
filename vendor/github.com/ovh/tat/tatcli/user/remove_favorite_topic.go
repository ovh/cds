package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserRemoveFavoriteTopic = &cobra.Command{
	Use:   "removeFavoriteTopic",
	Short: "Remove a favorite Topic: tatcli user removeFavoriteTopic <topicName>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserRemoveFavoriteTopic(args[0])
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user removeFavoriteTopic --help\n")
		}
	},
}
