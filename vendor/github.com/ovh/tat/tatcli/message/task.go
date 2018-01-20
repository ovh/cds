package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageTask = &cobra.Command{
	Use:   "task",
	Short: "Create a task from one message: tatcli message task /Private/username/tasks/sub-topic idMessage",
	Long: `Create a task from one message:
	tatcli message task /Private/username/tasks/sub-topic idMessage`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().MessageTask(args[0], args[1])
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to task a message: tatcli message task --help\n")
		}
	},
}
