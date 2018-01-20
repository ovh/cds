package user

import (
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserResetSystem = &cobra.Command{
	Use:   "resetSystemUser",
	Short: "Reset password for a system user (admin only): tatcli user resetSystemUser <username>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserResetSystem(tat.UsernameUserJSON{
				Username: args[0],
			})
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user resetSystemUser --help\n")
		}
	},
}
