package message

import (
	"strings"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageUpdate = &cobra.Command{
	Use:     "update",
	Aliases: []string{"up"},
	Short:   "Update a message (if it's enabled on topic): tatcli message update <topic> <idMessage> <my message...>",
	Long: `Update a message:
	tatcli message update <topic> <idMessage> <my message...>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 3 {
			topic := args[0]
			idMessage := args[1]
			message := strings.Join(args[2:], " ")
			out, err := internal.Client().MessageUpdate(topic, idMessage, message)
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to update a message: tatcli message update --help\n")
		}
	},
}
