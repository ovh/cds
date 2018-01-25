package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicAddRoGroup.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Rights RO recursively")
}

var cmdTopicAddRoGroup = &cobra.Command{
	Use:   "addRoGroup",
	Short: "Add Read Only Groups to a topic: tatcli topic addRoGroup [--recursive] <topic> <groupname1> [<groupname2>]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicAddRoGroups(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic addRoGroup --help\n")
		}
	},
}
