package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicAddRwUser.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Rights RW recursively")
}

var cmdTopicAddRwUser = &cobra.Command{
	Use:   "addRwUser",
	Short: "Add Read Write Users to a topic: tatcli topic addRwUser [--recursive] <topic> <username1> [username2]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicAddRwUsers(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic addRwUser --help\n")
		}
	},
}
