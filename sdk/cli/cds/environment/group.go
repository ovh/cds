package environment

import (
	"fmt"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
	"strconv"
)

// environmentGroupCmd Command to manage group management on environment
var environmentGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},

	Aliases: []string{"g"},
}

func init() {
	environmentGroupCmd.AddCommand(cmdEnvironmentAddGroup())
	environmentGroupCmd.AddCommand(cmdEnvironmentUpdateGroup())
	environmentGroupCmd.AddCommand(cmdEnvironmentRemoveGroup())
}

func cmdEnvironmentAddGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds environment group add <projectKey> <environmentName> <groupKey> <permission (4:read, 5:read+exec, 7:all)>",
		Long:  ``,
		Run:   addGroupInEnvironment,
	}
	return cmd
}

func addGroupInEnvironment(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	envName := args[1]
	groupName := args[2]
	permissionString := args[3]

	permissionInt, err := strconv.Atoi(permissionString)
	if err != nil {
		sdk.Exit("Permission should be an integer: %s.", err)
	}
	err = sdk.AddGroupInEnvironment(projectKey, envName, groupName, permissionInt)
	if err != nil {
		sdk.Exit("Error: cannot add group %s in environment %s (%s)\n", groupName, envName, err)
	}
	fmt.Printf("OK\n")
}

func cmdEnvironmentRemoveGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "cds environment group remove <projectKey> <environmentName> <groupKey>",
		Long:  ``,
		Run:   removeGroupFromEnvironment,
	}
	return cmd
}

func removeGroupFromEnvironment(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	envName := args[1]
	groupName := args[2]

	err := sdk.RemoveGroupFromEnvironment(projectKey, envName, groupName)
	if err != nil {
		sdk.Exit("Error: cannot remove group %s from environment %s (%s)\n", groupName, envName, err)
	}
	fmt.Printf("OK\n")
}

func cmdEnvironmentUpdateGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds environment group update <projectKey> <environmentName> <groupKey> <permission (4:read, 5:read+exec, 6:read+write, 7:all)>",
		Long:  ``,
		Run:   updateGroupInEnvironment,
	}
	return cmd
}

func updateGroupInEnvironment(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	envName := args[1]
	groupName := args[2]
	permissionString := args[3]

	permissionInt, err := strconv.Atoi(permissionString)
	if err != nil {
		sdk.Exit("Permission should be an integer: %s.", err)
	}
	err = sdk.UpdateGroupInEnvironment(projectKey, envName, groupName, permissionInt)
	if err != nil {
		sdk.Exit("Error: cannot update group permission in environment %s (%s)\n", envName, err)
	}
	fmt.Printf("OK\n")
}
