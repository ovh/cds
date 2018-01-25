package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageVoteDown = &cobra.Command{
	Use:   "votedown",
	Short: "Vote Down a message: tatcli message votedown <topic> <idMessage>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().MessageVoteDown(args[0], args[1])
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to votedown a message: tatcli message votedown --help\n")
		}
	},
}
