package group

import (
	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdGroupSetAdmin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setadmin",
		Short: "cds group setadmin <groupName> <user>",
		Long:  ``,
		Run:   setUserAdmin,
	}

	return cmd
}

func setUserAdmin(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	groupName := args[0]
	user := args[1]

	err := sdk.SetUserGroupAdmin(groupName, user)
	if err != nil {
		sdk.Exit("Error: Cannot set user %s admin of group %s (%s)\n", user, groupName, err)
	}
}

func cmdGroupUnsetAdmin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unsetadmin",
		Short: "cds group unsetadmin <groupName> <user>",
		Long:  ``,
		Run:   unsetUserAdmin,
	}

	return cmd
}

func unsetUserAdmin(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	groupName := args[0]
	user := args[1]

	err := sdk.UnsetUserGroupAdmin(groupName, user)
	if err != nil {
		sdk.Exit("Error: Cannot set user %s admin of group %s (%s)\n", user, groupName, err)
	}
}
