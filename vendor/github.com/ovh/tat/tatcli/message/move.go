package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageMove = &cobra.Command{
	Use:   "move",
	Short: "Move a message: tatcli message move <oldTopic> <idMessage> <newTopic>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 3 {
			out, err := internal.Client().MessageMove(args[0], args[1], args[2])
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to move a message: tatcli message move --help\n")
		}
	},
}
