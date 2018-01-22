package user

import (
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserReset = &cobra.Command{
	Use:   "reset",
	Short: "Ask for Reset a password: tatcli user reset <username> <email>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().UserReset(tat.UserResetJSON{
				Username: args[0],
				Email:    args[1],
				Callback: "tatcli --url=:scheme://:host::port:path user verify --save :username :token",
			})
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument to reset password: tatcli user reset --help\n")
		}
	},
}
