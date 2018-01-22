package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageLike = &cobra.Command{
	Use:   "like",
	Short: "Like a message: tatcli message like <topic> <idMessage>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().MessageLike(args[0], args[1])
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to like a message: tatcli message like --help\n")
		}
	},
}
