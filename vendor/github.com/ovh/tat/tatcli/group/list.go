package group

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdGroupList = &cobra.Command{
	Use:   "list",
	Short: "List all groups: tatcli group list <skip> <limit>",
	Run: func(cmd *cobra.Command, args []string) {
		skip, limit := internal.GetSkipLimit(args)
		out, err := internal.Client().GroupList(skip, limit)
		internal.Check(err)
		internal.Print(out)
	},
}
