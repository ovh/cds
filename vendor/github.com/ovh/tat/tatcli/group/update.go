package group

import (
	"strings"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdGroupUpdate = &cobra.Command{
	Use:   "update",
	Short: "update a group: tatcli group update <groupname> <newGroupname> <newDescription>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 3 {
			description := strings.Join(args[2:], " ")
			err := internal.Client().GroupUpdate(args[0], args[1], description)
			internal.Check(err)
		} else {
			internal.Exit("Invalid argument: tatcli group update --help\n")
		}
	},
}
