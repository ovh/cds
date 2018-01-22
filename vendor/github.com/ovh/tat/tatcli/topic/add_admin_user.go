package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicAddAdminUser.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Rights Admin recursively")
}

var cmdTopicAddAdminUser = &cobra.Command{
	Use:   "addAdminUser",
	Short: "Add Admin Users to a topic: tatcli topic addAdminUser [--recursive] <topic> <username1> [username2]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicAddAdminUsers(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic addAdminUser --help\n")
		}
	},
}
