package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicAddRwGroup.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Rights RW recursively")
}

var cmdTopicAddRwGroup = &cobra.Command{
	Use:   "addRwGroup",
	Short: "Add Read Write Groups to a topic: tatcli topic addRwGroup [--recursive] <topic> <groupname1> [<groupname2>]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicAddRwGroups(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic addRwGroup --help\n")
		}
	},
}
