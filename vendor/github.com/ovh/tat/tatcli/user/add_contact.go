package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserAddContact = &cobra.Command{
	Use:   "addContact",
	Short: "Add a contact: tatcli user addContact <contactUsername>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserAddContact(args[0])
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument to add contact: tatcli user addContact --help\n")
		}
	},
}
