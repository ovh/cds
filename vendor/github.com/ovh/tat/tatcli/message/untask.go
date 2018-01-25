package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageUntask = &cobra.Command{
	Use:   "untask",
	Short: "Remove a message from tasks: tatcli message untask /Private/username/tasks idMessage",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().MessageUntask(args[0], args[1])
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to untask a message: tatcli message untask --help\n")
		}
	},
}
