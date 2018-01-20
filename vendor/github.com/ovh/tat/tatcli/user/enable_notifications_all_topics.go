package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserEnableNotificationsAllTopics = &cobra.Command{
	Use:   "enableNotificationsAllTopics",
	Short: "Enable notifications on a topic: tatcli user enableNotificationsAllTopics",
	Run: func(cmd *cobra.Command, args []string) {
		out, err := internal.Client().UserEnableNotificationsAllTopics()
		internal.Check(err)
		internal.Print(out)
	},
}
