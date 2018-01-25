package user

import (
	"strconv"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserContacts = &cobra.Command{
	Use:   "contacts",
	Short: "Get contacts presences since n seconds: tatcli user contacts <seconds>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			seconds, err := strconv.Atoi(args[0])
			if err == nil {
				out, err := internal.Client().UserContacts(seconds)
				internal.Check(err)
				internal.Print(out)
				return
			}
		}
		internal.Exit("Invalid argument: tatcli user contacts --help\n")
	},
}
