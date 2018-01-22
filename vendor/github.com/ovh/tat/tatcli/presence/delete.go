package presence

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdPresenceDelete = &cobra.Command{
	Use:   "delete",
	Short: "Delete a user's presence on one topic: tatcli presence delete <topic> <username>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			err := internal.Client().PresenceDelete(args[0], args[1])
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli presence delete --help\n")
		}
	},
}
