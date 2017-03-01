package group

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var recursive bool

func cmdProjectAddGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds project group add <projectKey> <groupKey> <permission (4:read, 5:read+exec, 6:read+write, 7:all)>",
		Long:  ``,
		Run:   addGroupInProject,
	}
	return cmd
}

func addGroupInProject(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	groupName := args[1]
	permissionString := args[2]

	permissionInt, err := strconv.Atoi(permissionString)
	if err != nil {
		sdk.Exit("Permission should be an integer: %s.", err)
	}
	err = sdk.AddGroupInProject(projectKey, groupName, permissionInt)
	if err != nil {
		sdk.Exit("Error: cannot add group %s in project %s (%s)\n", groupName, projectKey, err)
	}
	fmt.Printf("OK\n")
}
