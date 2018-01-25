package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicDeleteRoGroup.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Delete Rights RO recursively")
}

var cmdTopicDeleteRoGroup = &cobra.Command{
	Use:   "deleteRoGroup",
	Short: "Delete Read Only Groups from a topic: tatcli topic deleteRoGroup [--recursive] <topic> <groupname1> [<groupname2>]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicDeleteRoGroups(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic deleteRoGroup --help\n")
		}
	},
}
