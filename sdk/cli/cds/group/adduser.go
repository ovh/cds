package group

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdGroupAddUser() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adduser",
		Short: "cds group adduser <groupName> <user1> <user2>",
		Long:  ``,
		Run:   addUser,
	}

	return cmd
}

func addUser(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	groupName := args[0]
	users := args[1:len(args)]

	err := sdk.AddUsersInGroup(groupName, users)
	if err != nil {
		sdk.Exit("%s\n", err)
	}
	fmt.Printf("OK\n")
}
