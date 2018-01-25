package user

import (
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserSetAdmin = &cobra.Command{
	Use:   "setAdmin",
	Short: "Grant user to Tat admin (admin only): tatcli user setAdmin <username>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			out, err := internal.Client().UserSetAdmin(tat.UsernameUserJSON{
				Username: args[0],
			})
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user setAdmin --help\n")
		}
	},
}
