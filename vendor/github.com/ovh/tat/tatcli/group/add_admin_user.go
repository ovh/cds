package group

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdGroupAddAdminUser = &cobra.Command{
	Use:   "addAdminUser",
	Short: "Add Admin Users to a group: tacli group addAdminUser <groupname> <username1> [<username2> ... ]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			err := internal.Client().GroupAddAdminUsers(args[0], args[1:])
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli group addAdminUser --help\n")
		}
	},
}
