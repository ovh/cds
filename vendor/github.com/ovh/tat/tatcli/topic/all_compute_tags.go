package topic

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdTopicAllComputeTags = &cobra.Command{
	Use:   "allcomputetags",
	Short: "Compute Tags on all topics, only for tat admin : tatcli topic allcomputetags",
	Run: func(cmd *cobra.Command, args []string) {
		out, err := internal.Client().TopicAllComputeTags()
		internal.Check(err)
		if internal.Verbose {
			internal.Print(out)
		}
	},
}
