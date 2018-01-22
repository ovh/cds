package user

import (
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var withGroups bool

func init() {
	cmdUserList.Flags().BoolVarP(&withGroups, "with-groups", "g", false, "List Users with groups, admin only")
}

var cmdUserList = &cobra.Command{
	Use:   "list",
	Short: "List all users: tatcli user list [<skip>] [<limit>]",
	Run: func(cmd *cobra.Command, args []string) {
		skip, limit := internal.GetSkipLimit(args)
		criteria := &tat.UserCriteria{
			Skip:       skip,
			Limit:      limit,
			WithGroups: withGroups,
		}
		out, err := internal.Client().UserList(criteria)
		internal.Check(err)
		internal.Print(out)
	},
}
