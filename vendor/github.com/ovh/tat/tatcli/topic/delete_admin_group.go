package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicDeleteAdminGroup.Flags().BoolVarP(&recursive, "recursive", "r", false, "Apply Delete Rights Admin recursively")
}

var cmdTopicDeleteAdminGroup = &cobra.Command{
	Use:   "deleteAdminGroup",
	Short: "Delete Admin Groups from a topic: tatcli topic deleteAdminGroup [--recursive] <topic> <groupname1> [<groupname2>]...",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicDeleteAdminGroups(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic deleteAdminGroup --help\n")
		}
	},
}
