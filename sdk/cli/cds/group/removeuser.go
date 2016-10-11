package group

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdGroupRemoveUser() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "removeuser",
		Short: "cds group removeuser <groupName> <user>",
		Long:  ``,
		Run:   removeUser,
	}

	return cmd
}

func removeUser(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	groupName := args[0]
	user := args[1]

	err := sdk.RemoveUserFromGroup(groupName, user)
	if err != nil {
		sdk.Exit("%s\n", err)
	}
	fmt.Printf("OK\n")
}
