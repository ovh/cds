package group

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdGroupAddUser = &cobra.Command{
	Use:   "addUser",
	Short: "Add Users to a group: tacli group addUser <groupname> <username1> [<username2> ... ]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().GroupAddUsers(args[0], args[1:])
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli group addUser --help\n")
		}
	},
}
