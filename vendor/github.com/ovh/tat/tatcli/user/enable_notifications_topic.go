package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserEnableNotificationsTopic = &cobra.Command{
	Use:   "enableNotificationsTopic",
	Short: "Enable notifications on a topic: tatcli user enableNotificationsTopic <topicName>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserEnableNotificationsTopic(args[0])
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user enableNotificationsTopic --help\n")
		}
	},
}
