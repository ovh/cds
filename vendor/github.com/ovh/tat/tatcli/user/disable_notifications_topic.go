package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserDisableNotificationsTopic = &cobra.Command{
	Use:   "disableNotificationsTopic",
	Short: "Disable notifications on a topic: tatcli user disableNotificationsTopic <topicName>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserDisableNotificationsTopic(args[0])
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user disableNotificationsTopic --help\n")
		}
	},
}
