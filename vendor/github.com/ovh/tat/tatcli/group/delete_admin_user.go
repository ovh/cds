package group

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdGroupDeleteAdminUser = &cobra.Command{
	Use:   "deleteAdminUser",
	Short: "Delete Admin Users from a group: tacli group deleteAdminUser <groupname> <username1> [<username2> ... ]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().GroupDeleteAdminUsers(args[0], args[1:])
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli group deleteAdminUser --help\n")
		}
	},
}
