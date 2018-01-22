package message

import (
	"strings"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageUnlabel = &cobra.Command{
	Use:   "unlabel",
	Short: "Remove a label from a message: tatcli message unlabel <topic> <idMessage> <my Label>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 3 {
			label := strings.Join(args[2:], " ")
			out, err := internal.Client().MessageUnlabel(args[0], args[1], label)
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to unlabel a message: tatcli message unlabel --help\n")
		}
	},
}
