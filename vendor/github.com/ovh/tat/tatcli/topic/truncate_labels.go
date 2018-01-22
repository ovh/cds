package topic

import (
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdTopicTruncateLabels = &cobra.Command{
	Use:   "truncatelabels",
	Short: "Truncate Labels on this topic, only for tat admin and administrators on topic : tatcli topic truncatelabels <topic>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().TopicTruncateLabels(tat.TopicNameJSON{Topic: args[0]})
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument: tatcli topic truncatelabels --help\n")
		}
	},
}
