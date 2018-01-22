package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageUnVoteUP = &cobra.Command{
	Use:   "unvoteup",
	Short: "Remove a vote UP from a message: tatcli message unvoteup <topic> <idMessage>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().MessageUnVoteUP(args[0], args[1])
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to unvoteup a message: tatcli message unvoteup --help\n")
		}
	},
}
