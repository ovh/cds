package group

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdProjectRemoveGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "cds project group remove <projectKey> <groupKey>",
		Long:  ``,
		Run:   removeGroupInProject,
	}
	return cmd
}

func removeGroupInProject(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	groupName := args[1]

	err := sdk.RemoveGroupFromProject(projectKey, groupName)
	if err != nil {
		sdk.Exit("Error: cannot remove group %s from project %s (%s)\n", groupName, projectKey, err)
	}
	fmt.Printf("OK\n")
}
