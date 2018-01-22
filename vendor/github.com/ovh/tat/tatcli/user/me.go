package user

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserMe = &cobra.Command{
	Use:   "me",
	Short: "Get Information about you: tatcli user me",
	Run: func(cmd *cobra.Command, args []string) {
		out, err := internal.Client().UserMe()
		internal.Check(err)
		internal.Print(out)
	},
}
