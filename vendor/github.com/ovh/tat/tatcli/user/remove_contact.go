package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserRemoveContact = &cobra.Command{
	Use:   "removeContact",
	Short: "Remove a contact: tatcli user removeContact <contactUsername>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserRemoveContact(args[0])
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user removeContact --help\n")
		}
	},
}
