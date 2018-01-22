package message

import (
	"strings"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageConcat = &cobra.Command{
	Use:   "concat",
	Short: "Update a message (if it's enabled on topic) by adding additional text at the end of message: tatcli message concat <topic> <idMessage> <additional text...>",
	Long: `Update a message:
	It could be used to add tag or text at the end of one message.
	tatcli message concat <topic> <idMessage> <additional text...>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 3 {
			topic := args[0]
			idMessage := args[1]
			addText := strings.Join(args[2:], " ")
			out, err := internal.Client().MessageConcat(topic, idMessage, addText)
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to update a message: tatcli message concat --help\n")
		}
	},
}
