package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicDeleteParameter.Flags().BoolVarP(&recursive, "recursive", "r", false, "Remove Parameter recursively")
}

var cmdTopicDeleteParameter = &cobra.Command{
	Use:   "deleteParameter",
	Short: "Remove Parameter to a topic: tatcli topic deleteParameter [--recursive] <topic> <key> [<key2>]... ",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().TopicDeleteParameters(args[0], args[1:], recursive)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli topic deleteParameter --help\n")
		}
	},
}
