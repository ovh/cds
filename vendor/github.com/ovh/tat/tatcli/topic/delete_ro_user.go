package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicDeleteRoUser.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Delete Rights RO recursively")
}

var cmdTopicDeleteRoUser = &cobra.Command{
	Use:   "deleteRoUser",
	Short: "Delete Read Only Users from a topic: tatcli topic deleteRoUser [--recursive] <topic> <username1> [username2]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicDeleteRoUsers(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic deleteRoUser --help\n")
		}
	},
}
