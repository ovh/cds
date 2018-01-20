package group

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdGroupDelete = &cobra.Command{
	Use:   "delete",
	Short: "delete a group: tatcli group delete <groupname>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			internal.Client().GroupDelete(args[0])
		} else {
			internal.Exit("Invalid argument: tatcli group delete --help\n")
		}
	},
}
