package application

import (
	"fmt"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
	"strconv"
)

// applicationGroupCmd Command to manage group management on application
var applicationGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},

	Aliases: []string{"g"},
}

func init() {
	applicationGroupCmd.AddCommand(cmdApplicationAddGroup())
	applicationGroupCmd.AddCommand(cmdApplicationUpdateGroup())
	applicationGroupCmd.AddCommand(cmdApplicationRemoveGroup())
}

func cmdApplicationAddGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds application group add <projectKey> <applicationName> <groupKey> <permission (4:read, 5:read+exec, 7:all)>",
		Long:  ``,
		Run:   addGroupInApplication,
	}
	return cmd
}

func addGroupInApplication(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	groupName := args[2]
	permissionString := args[3]

	permissionInt, err := strconv.Atoi(permissionString)
	if err != nil {
		sdk.Exit("Permission should be an integer: %s.", err)
	}
	err = sdk.AddGroupInApplication(projectKey, appName, groupName, permissionInt)
	if err != nil {
		sdk.Exit("Error: cannot add group %s in application %s (%s)\n", groupName, appName, err)
	}
	fmt.Printf("OK\n")
}

func cmdApplicationRemoveGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "cds application group remove <projectKey> <applicationName> <groupKey>",
		Long:  ``,
		Run:   removeGroupFromApplication,
	}
	return cmd
}

func removeGroupFromApplication(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	groupName := args[2]

	err := sdk.RemoveGroupFromApplication(projectKey, appName, groupName)
	if err != nil {
		sdk.Exit("Error: cannot remove group %s from application %s (%s)\n", groupName, appName, err)
	}
	fmt.Printf("OK\n")
}

func cmdApplicationUpdateGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds application group update <projectKey> <applicationName> <groupKey> <permission (4:read, 5:read+exec, 6:read+write, 7:all)>",
		Long:  ``,
		Run:   updateGroupInApplication,
	}
	return cmd
}

func updateGroupInApplication(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	groupName := args[2]
	permissionString := args[3]

	permissionInt, err := strconv.Atoi(permissionString)
	if err != nil {
		sdk.Exit("Permission should be an integer: %s.", err)
	}
	err = sdk.UpdateGroupInApplication(projectKey, appName, groupName, permissionInt)
	if err != nil {
		sdk.Exit("Error: cannot update group permission in application %s (%s)\n", appName, err)
	}
	fmt.Printf("OK\n")
}
