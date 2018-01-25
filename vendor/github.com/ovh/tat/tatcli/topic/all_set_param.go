package topic

import (
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdTopicAllSetParam = &cobra.Command{
	Use:   "allsetparam",
	Short: "Set a param for all topics, only for tat admin : tatcli topic allsetparam <paramName> <paramValue>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().TopicAllSetParam(tat.ParamJSON{ParamName: args[0], ParamValue: args[1]})
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument: tatcli topic allsetparam --help\n")
		}
	},
}
