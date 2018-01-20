package user

import (
	"strings"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserAdd = &cobra.Command{
	Use:   "add",
	Short: "Add a user: tatcli user add <username> <email> <fullname>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 3 {
			out, err := internal.Client().UserAdd(tat.UserCreateJSON{
				Username: args[0],
				Fullname: strings.Join(args[2:], " "),
				Email:    args[1],
				Callback: "tatcli --url=:scheme://:host::port:path user verify --save :username :token",
			})
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument to add user: tatcli user add --help\n")
		}
	},
}
