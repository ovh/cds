package pipeline

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var pipelineStageCmd = &cobra.Command{
	Use: "stage",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	pipelineStageCmd.AddCommand(cmdPipelineAddStage())
	pipelineStageCmd.AddCommand(pipelineDeleteStageCmd())
	pipelineStageCmd.AddCommand(pipelineMoveStageCmd())
	pipelineStageCmd.AddCommand(pipelineRenameStageCmd())
	pipelineStageCmd.AddCommand(pipelineChangeStateStageCmd())
}

func cmdPipelineAddStage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds pipeline stage add <projectKey> <pipelineName> <stageName>",
		Run:   addStage,
	}

	return cmd
}

func addStage(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	stageName := strings.Join(args[2:len(args)], " ")
	err := sdk.AddStage(projectKey, pipelineName, stageName)
	if err != nil {
		sdk.Exit("Error: cannot add stage %s (%s)\n", stageName, err)
	}
	fmt.Printf("OK\n")
}

func pipelineDeleteStageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "cds pipeline stage delete <projectKey> <pipelineName> <pipelineStageID>",
		Run:   deleteStage,
	}
	return cmd
}

func deleteStage(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	pipelineStageID := args[2]
	err := sdk.DeleteStage(projectKey, pipelineName, pipelineStageID)
	if err != nil {
		sdk.Exit("Error: cannot delete stage (%s)\n", err)
	}
	fmt.Printf("Stage deleted.\n")
}

func pipelineChangeStateStageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "cds pipeline stage state <projectKey> <pipelineName> <pipelineStageID> <state[true/false]>",
		Run:   changeStateStage,
	}
	return cmd
}

func changeStateStage(cmd *cobra.Command, args []string) {
	if len(args) < 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	pipelineStageIDString := args[2]
	stateString := args[3]

	state, err := strconv.ParseBool(stateString)
	if err != nil {
		sdk.Exit("Error: state must be a boolean(%s)\n", err)
	}
	err = sdk.ChangeStageState(projectKey, pipelineName, pipelineStageIDString, state)
	if err != nil {
		sdk.Exit("Error: cannot enable/disable stage (%s)\n", err)
	}
	fmt.Printf("Stage updated.\n")
}

func pipelineRenameStageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename",
		Short: "cds pipeline stage rename <projectKey> <pipelineName> <pipelineStageID> <stageName>",
		Run:   renameStage,
	}
	return cmd
}

func renameStage(cmd *cobra.Command, args []string) {
	if len(args) < 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	pipelineStageIDString := args[2]
	newName := strings.Join(args[3:len(args)], " ")

	err := sdk.RenameStage(projectKey, pipelineName, pipelineStageIDString, newName)
	if err != nil {
		sdk.Exit("Error: cannot rename stage (%s)\n", err)
	}
	fmt.Printf("Stage renamed.\n")
}

func pipelineMoveStageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move",
		Short: "cds pipeline stage move <projectKey> <pipelineName> <pipelineStageID> <order>",
		Run:   moveStage,
	}
	return cmd
}

func moveStage(cmd *cobra.Command, args []string) {
	if len(args) < 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	pipelineStageIDString := args[2]
	orderString := args[3]

	pipelineStageID, err := strconv.ParseInt(pipelineStageIDString, 10, 64)
	if err != nil {
		sdk.Exit("Error: pipelineStageID must be a int (%s)\n", err)
	}

	order, err := strconv.Atoi(orderString)
	if err != nil {
		sdk.Exit("Error: order must be a int (%s)\n", err)
	}

	err = sdk.MoveStage(projectKey, pipelineName, pipelineStageID, order)
	if err != nil {
		sdk.Exit("Error: cannot move stage (%s)\n", err)
	}
	fmt.Printf("Stage moved.\n")
}
