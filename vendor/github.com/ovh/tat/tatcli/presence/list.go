package presence

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdPresenceList = &cobra.Command{
	Use:   "list",
	Short: "List all presences on one topic: tatcli presence list <topic> [<skip>] [<limit>]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			skip, limit := internal.GetSkipLimit(args)
			out, err := internal.Client().PresenceList(args[0], skip, limit)
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli presence list --help\n")
		}
	},
}
