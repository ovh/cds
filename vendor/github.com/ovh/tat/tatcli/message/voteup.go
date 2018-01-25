package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageVoteUP = &cobra.Command{
	Use:   "voteup",
	Short: "Vote UP a message: tatcli message voteup <topic> <idMessage>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().MessageVoteUP(args[0], args[1])
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to voteup a message: tatcli message voteup --help\n")
		}
	},
}
