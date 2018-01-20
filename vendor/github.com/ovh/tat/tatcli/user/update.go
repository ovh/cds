package user

import (
	"strings"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserUpdate = &cobra.Command{
	Use:   "update",
	Short: "Update Fullname and Email of a user (admin only): tatcli user update <username> <newEmail> <newFullname>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 3 {
			out, err := internal.Client().UserUpdate(tat.UpdateUserJSON{
				Username:    args[0],
				NewEmail:    args[1],
				NewFullname: strings.Join(args[2:], " "),
			})
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user update --help\n")
		}
	},
}
