package presence

import (
	"strings"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdPresenceAdd = &cobra.Command{
	Use:   "add",
	Short: "Add a new presence on one topic with status (online, offline, busy): tatcli presence add <topic> <status>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			status := strings.Join(args[1:], " ")
			out, err := internal.Client().PresenceAddAndGet(args[0], status)
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument: tatcli presence add --help\n")
		}
	},
}
