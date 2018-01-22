package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicAddRoUser.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Rights RO recursively")
}

var cmdTopicAddRoUser = &cobra.Command{
	Use:   "addRoUser",
	Short: "Add Read Only Users to a topic: tatcli topic addRoUser [--recursive] <topic> <username1> [username2]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicAddRoUsers(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic addRoUser --help\n")
		}
	},
}
