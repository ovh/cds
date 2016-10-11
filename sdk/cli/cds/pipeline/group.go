package pipeline

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/ovh/cds/sdk"
	"strconv"
)

// CmdGroup Command to manage group management on project
var pipelineGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},

	Aliases: []string{"g"},
}

func init() {
	pipelineGroupCmd.AddCommand(cmdPipelineAddGroup())
	pipelineGroupCmd.AddCommand(cmdPipelineUpdateGroup())
	pipelineGroupCmd.AddCommand(cmdPipelineRemoveGroup())
}

func cmdPipelineAddGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds pipeline group add <projectKey> <pipelineName> <groupKey> <permission (4:read, 5:read+exec, 7:all)>",
		Long:  ``,
		Run:   addGroupInPipeline,
	}
	return cmd
}

func addGroupInPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	groupName := args[2]
	permissionString := args[3]

	permissionInt, err := strconv.Atoi(permissionString)
	if err != nil {
		sdk.Exit("Permission should be an integer: %s.", err)
	}
	err = sdk.AddGroupInPipeline(projectKey, pipelineName, groupName, permissionInt)
	if err != nil {
		sdk.Exit("Error: cannot add group %s in pipelineName %s (%s)\n", groupName, pipelineName, err)
	}
	fmt.Printf("OK\n")
}

func cmdPipelineRemoveGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "cds pipeline group remove <projectKey> <pipelineName> <groupKey>",
		Long:  ``,
		Run:   removeGroupFromPipeline,
	}
	return cmd
}

func removeGroupFromPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	groupName := args[2]

	err := sdk.RemoveGroupFromPipeline(projectKey, pipelineName, groupName)
	if err != nil {
		sdk.Exit("Error: cannot remove group %s from pipeline %s (%s)\n", groupName, pipelineName, err)
	}
	fmt.Printf("OK\n")
}

func cmdPipelineUpdateGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds pipeline group update <projectKey> <pipelineName> <groupKey> <permission (4:read, 5:read+exec, 6:read+write, 7:all)>",
		Long:  ``,
		Run:   updateGroupInPipeline,
	}
	return cmd
}

func updateGroupInPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	groupName := args[2]
	permissionString := args[3]

	permissionInt, err := strconv.Atoi(permissionString)
	if err != nil {
		sdk.Exit("Permission should be an integer: %s.", err)
	}
	err = sdk.UpdateGroupInPipeline(projectKey, pipelineName, groupName, permissionInt)
	if err != nil {
		sdk.Exit("Error: cannot update group permission in pipeline %s (%s)\n", pipelineName, err)
	}
	fmt.Printf("OK\n")
}
