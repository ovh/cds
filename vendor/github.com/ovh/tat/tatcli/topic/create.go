package topic

import (
	"strings"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdTopicCreate = &cobra.Command{
	Use:   "create",
	Short: "Create a new topic: tatcli create <topic> <description of topic>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			description := strings.Join(args[1:], " ")
			topic, err := internal.Client().TopicCreate(tat.TopicCreateJSON{
				Topic:       args[0],
				Description: description,
			})
			internal.Check(err)
			if internal.Verbose {
				internal.Print(topic)
			}
		} else {
			internal.Exit("Invalid argument: tatcli topic create --help\n")
		}
	},
}
