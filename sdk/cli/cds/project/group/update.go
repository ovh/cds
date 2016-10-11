package group

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/ovh/cds/sdk"
)

func cmdProjectUpdateGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds project group update <projectKey> <groupKey> <permission (4:read, 5:read+exec, 6:read+write, 7:all)>",
		Long:  ``,
		Run:   updateGroupInProject,
	}
	return cmd
}

func updateGroupInProject(cmd *cobra.Command, args []string) {
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
	err = sdk.UpdateGroupInProject(projectKey, groupName, permissionInt)
	if err != nil {
		sdk.Exit("Error: cannot update group permission in project %s (%s)\n", projectKey, err)
	}
	fmt.Printf("OK\n")
}
