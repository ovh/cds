package user

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

var cmdUserListQuiet bool

func cmdUserList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "",
		Long:    ``,
		Aliases: []string{"ls"},
		Run:     listUser,
	}

	cmd.Flags().BoolVarP(&cmdUserListQuiet, "quiet", "q", false, "")

	return cmd
}

func listUser(cmd *cobra.Command, args []string) {
	users, err := sdk.ListUsers()
	if err != nil {
		sdk.Exit("Error: cannot list user (%s)\n", err)
	}

	for i := range users {
		if cmdUserListQuiet {
			fmt.Printf("%s\n", users[i].Username)
			continue
		}
		fmt.Printf("%s %s %s\n", users[i].Username, users[i].Email, users[i].Fullname)
	}
}
