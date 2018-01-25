package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageUnVoteDown = &cobra.Command{
	Use:   "unvotedown",
	Short: "Remove a vote down from a message: tatcli message unvotedown <topic> <idMessage>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().MessageUnVoteDown(args[0], args[1])
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to unvotedown a message: tatcli message unvotedown --help\n")
		}
	},
}
