package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdTopicAllComputeLabels = &cobra.Command{
	Use:   "allcomputelabels",
	Short: "Compute Labels on all topics, only for tat admin : tatcli topic allcomputelabels",
	Run: func(cmd *cobra.Command, args []string) {
		out, err := internal.Client().TopicAllComputeLabels()
		internal.Check(err)
		if internal.Verbose {
			internal.Print(out)
		}
	},
}
