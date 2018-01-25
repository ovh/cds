package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdTopicAllComputeReplies = &cobra.Command{
	Use:   "allcomputereplies",
	Short: "Compute Replies on all topics, only for tat admin : tatcli topic allcomputereplies",
	Run: func(cmd *cobra.Command, args []string) {
		out, err := internal.Client().TopicAllComputeReplies()
		internal.Check(err)
		if internal.Verbose {
			internal.Print(out)
		}
	},
}
