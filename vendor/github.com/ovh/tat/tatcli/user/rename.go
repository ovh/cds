package user

import (
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserRename = &cobra.Command{
	Use:   "rename",
	Short: "Rename username of a user (admin only): tatcli user rename <oldUsername> <newUsername>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().UserRename(tat.RenameUserJSON{
				Username:    args[0],
				NewUsername: args[1],
			})
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user rename --help\n")
		}
	},
}
