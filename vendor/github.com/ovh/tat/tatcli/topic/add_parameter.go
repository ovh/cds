package topic

import (
	"strings"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicAddParameter.Flags().BoolVarP(&recursive, "recursive", "r", false, "Add Parameter recursively")
}

var cmdTopicAddParameter = &cobra.Command{
	Use:   "addParameter",
	Short: "Add Parameter to a topic: tatcli topic addParameter [--recursive] <topic> <key>:<value> [<key2>:<value2>]... ",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			topicAddParameter(args[0], args[1:])
		} else {
			internal.Exit("Invalid argument: tatcli topic addParameter --help\n")
		}
	},
}

func topicAddParameter(topic string, parameters []string) {
	for _, param := range parameters {
		parameterSplitted := strings.Split(param, ":")
		if len(parameterSplitted) != 2 {
			continue
		}
		_, err := internal.Client().TopicAddParameter(topic, parameterSplitted[0], parameterSplitted[1], recursive)
		internal.Check(err)
	}
}
