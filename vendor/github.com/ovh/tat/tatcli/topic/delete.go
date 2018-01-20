package topic

import (
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdTopicDelete = &cobra.Command{
	Use:   "delete",
	Short: "Delete a topic: tatcli delete <topic>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().TopicDelete(tat.TopicNameJSON{Topic: args[0]})
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli topic delete --help\n")
		}
	},
}
