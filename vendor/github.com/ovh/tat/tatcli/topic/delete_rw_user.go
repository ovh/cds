package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicDeleteRwUser.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Delete Rights RW recursively")
}

var cmdTopicDeleteRwUser = &cobra.Command{
	Use:   "deleteRwUser",
	Short: "Delete Read Write Users from a topic: tatcli topic deleteRwUser [--recursive] <topic> <username1> [username2]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicDeleteRwUsers(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic deleteRwUser --help\n")
		}
	},
}
