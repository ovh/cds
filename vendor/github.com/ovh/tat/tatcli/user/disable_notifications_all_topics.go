package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserDisableNotificationsAllTopics = &cobra.Command{
	Use:   "disableNotificationsAllTopics",
	Short: "Disable notifications on all topics: tatcli user disableNotificationsAllTopics",
	Run: func(cmd *cobra.Command, args []string) {
		out, err := internal.Client().UserDisableNotificationsAllTopics()
		internal.Check(err)
		internal.Print(out)
	},
}
