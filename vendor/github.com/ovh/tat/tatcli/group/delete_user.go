package group

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdGroupDeleteUser = &cobra.Command{
	Use:   "deleteUser",
	Short: "Delete Users from a group: tacli group deleteUser <groupname> <username1> [<username2> ... ]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().GroupDeleteUsers(args[0], args[1:])
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli group deleteUser --help\n")
		}
	},
}
