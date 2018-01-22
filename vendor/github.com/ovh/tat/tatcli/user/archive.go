package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserArchive = &cobra.Command{
	Use:   "archive",
	Short: "Archive a user (admin only): tatcli user archive <username>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			err := internal.Client().UserArchive(args[0])
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli user archive --help\n")
		}
	},
}
