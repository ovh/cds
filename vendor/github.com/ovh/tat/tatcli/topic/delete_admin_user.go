package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicDeleteAdminUser.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Delete Rights Admin recursively")
}

var cmdTopicDeleteAdminUser = &cobra.Command{
	Use:   "deleteAdminUser",
	Short: "Delete Admin Users from a topic: tatcli topic deleteAdminUser [--recursive] <topic> <username1> [username2]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicDeleteAdminGroups(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic deleteAdminUser --help\n")
		}
	},
}
