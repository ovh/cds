package topic

import (
	"fmt"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var force bool

func init() {
	cmdTopicTruncate.Flags().BoolVarP(&force, "force", "", false, "--force : truncate without asking confirmation")
}

var cmdTopicTruncate = &cobra.Command{
	Use:   "truncate",
	Short: "Remove all messages in a topic, only for tat admin and administrators on topic : tatcli topic truncate <topic> [--force]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			topicTruncate(args[0])
		} else {
			internal.Exit("Invalid argument: tatcli topic truncate --help\n")
		}
	},
}

func topicTruncate(topic string) {
	j := tat.TopicNameJSON{Topic: topic}
	if force {
		out, err := internal.Client().TopicTruncate(j)
		internal.Check(err)
		if internal.Verbose {
			internal.Print(out)
		}
	} else {
		fmt.Print("Are you really sure ? You will delete all messages even if a user has a message in his tasks. Please enter again topic name to confirm: ")
		var confirmTopic string
		fmt.Scanln(&confirmTopic)

		if confirmTopic == topic {
			fmt.Printf("Please enter 'yes' to confirm removing all messages from %s: ", topic)
			var confirmYes string
			fmt.Scanln(&confirmYes)
			if confirmYes == "yes" {
				out, err := internal.Client().TopicTruncate(j)
				internal.Check(err)
				internal.Print(out)
				return
			}
		} else {
			fmt.Printf("Error. You enter %s instead of %s\n", confirmTopic, topic)
		}
		fmt.Println("Nothing done")
	}
}
