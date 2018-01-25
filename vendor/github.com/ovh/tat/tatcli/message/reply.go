package message

import (
	"strings"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageReply = &cobra.Command{
	Use:     "reply",
	Aliases: []string{"r"},
	Short:   "Reply to a message: tatcli message reply <topic> <inReplyOfId> <my message...>",
	Long: `Reply to a message:
	tatcli message reply <topic> <inReplyOfId> <my message...>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 3 {
			topic := args[0]
			inReplyOfID := args[1]
			message := strings.Join(args[2:], " ")
			out, err := internal.Client().MessageReply(topic, inReplyOfID, message)
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to reply to a message: tatcli message reply --help\n")
		}
	},
}
