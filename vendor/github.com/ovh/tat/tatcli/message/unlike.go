package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageUnlike = &cobra.Command{
	Use:   "unlike",
	Short: "Unlike a message: tatcli message unlike <topic> <idMessage>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().MessageUnlike(args[0], args[1])
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to unlike a message: tatcli message unlike --help\n")
		}
	},
}
