package user

import (
	"fmt"
	"strconv"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdUserCheck = &cobra.Command{
	Use:   "check",
	Short: "Check Private Topics and Default Group on one user (admin only): tatcli user check <username> <fixPrivateTopics> <fixDefaultGroup>",
	Long: `Check Private Topics and Default Group on one user:
tatcli user check <username> <fixPrivateTopics> <fixDefaultGroup>

Example :

tatcli check username true true
		`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 3 {
			fixPrivateTopics, e1 := strconv.ParseBool(args[1])
			fixDefaultGroup, e2 := strconv.ParseBool(args[2])
			if e1 != nil || e2 != nil {
				fmt.Println("Invalid argument: tatcli user check --help")
			}
			out, err := internal.Client().UserCheck(tat.CheckTopicsUserJSON{
				Username:         args[0],
				FixPrivateTopics: fixPrivateTopics,
				FixDefaultGroup:  fixDefaultGroup,
			})
			internal.Check(err)
			internal.Print(out)
		} else {
			internal.Exit("Invalid argument: tatcli user check --help\n")
		}
	},
}
