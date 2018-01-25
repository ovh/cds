package group

import (
	"strings"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdGroupCreate = &cobra.Command{
	Use:   "create",
	Short: "create a new group: tatcli group create <groupname> <description>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			description := strings.Join(args[1:], " ")
			g, err := internal.Client().GroupCreate(tat.GroupJSON{Name: args[0], Description: description})
			internal.Check(err)
			internal.Print(g)
		} else {
			internal.Exit("Invalid argument: tatcli group create --help\n")
		}
	},
}
