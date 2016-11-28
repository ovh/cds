package pipeline

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var cmdPipelineAddActionArguments []string
var cmdPipelineAddActionStageNumber string

var pipelineActionCmd = &cobra.Command{
	Use:   "action",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	pipelineActionCmd.AddCommand(pipelineAddActionCmd())
	pipelineActionCmd.AddCommand(pipelineDeleteActionCmd())
	pipelineActionCmd.AddCommand(pipelineMoveActionCmd())
}

func pipelineMoveActionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move",
		Short: "cds pipeline action move <projectKey> <pipelineName> <actionPipelineID> <newOrder>",
		Long:  ``,
		Run:   movePipelineAction,
	}

	return cmd
}

func movePipelineAction(cmd *cobra.Command, args []string) {

	if len(args) < 4 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	pipelineName := args[1]
	actionIDString := args[2]
	newOrderString := args[3]

	actionID, err := strconv.ParseInt(actionIDString, 10, 64)
	if err != nil {
		sdk.Exit("Error: actionPipelineID must be an integer (%s)\n", err)
	}
	newOrder, err := strconv.Atoi(newOrderString)
	if err != nil {
		sdk.Exit("Error: newOrder must be an integer (%s)\n", err)
	}

	err = sdk.MoveActionInPipeline(projectKey, pipelineName, actionID, newOrder)
	if err != nil {
		sdk.Exit("Error: cannot move action (%s)\n", err)
	}

	fmt.Printf("Action moved\n")
}

func pipelineAddActionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds pipeline action add <projectKey> <pipelineName> <actionName> [-p PARAMETER] [--stage=buildOrder]",
		Long:  ``,
		Run:   addPipelineAction,
	}

	cmd.Flags().StringVarP(&cmdPipelineAddActionStageNumber, "stage", "", "0", "Stage number")
	cmd.Flags().StringSliceVarP(&cmdPipelineAddActionArguments, "parameter", "p", nil, "Action parameters")
	return cmd
}

func addPipelineAction(cmd *cobra.Command, args []string) {

	if len(args) < 3 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	pipelineName := args[1]
	actionName := args[2]
	actionArgs := cmdPipelineAddActionArguments
	pipelineStageNumberS := cmdPipelineAddActionStageNumber

	pipelineStageNumber, err := strconv.Atoi(pipelineStageNumberS)
	if err != nil {
		sdk.Exit("Error: stage number is not a valid number (%s)\n", err)
	}

	// Get pipeline to get stage IDs
	var pipelineStageID int64
	pipeline, err := sdk.GetPipeline(projectKey, pipelineName)
	if err != nil {
		sdk.Exit("Error: cannot retrieve pipeline %s (%s)\n", pipelineName, err)
	}
	for i := range pipeline.Stages {
		if i+1 == pipelineStageNumber {
			pipelineStageID = pipeline.Stages[i].ID
		}
	}

	parameters := make([]sdk.Parameter, len(actionArgs))
	for index, elt := range actionArgs {
		argSplitted := strings.SplitN(elt, "=", 2)

		p := sdk.Parameter{
			Name:  argSplitted[0],
			Value: argSplitted[1],
		}
		parameters[index] = p
	}

	joined, err := sdk.NewJoinedAction(actionName, parameters)
	joined.Enabled = true
	if err != nil {
		sdk.Exit("Error: cannot create joined action (%s)\n", err)
	}

	err = sdk.AddJoinedAction(projectKey, pipelineName, pipelineStageID, joined)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	fmt.Printf("Action %s added to pipeline %s\n", actionName, pipelineName)
}

func pipelineDeleteActionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "cds pipeline action delete <projectKey> <pipelineName> <actionPipelineID>",
		Long:  ``,
		Run:   deletePipelineAction,
	}
	return cmd
}

func deletePipelineAction(cmd *cobra.Command, args []string) {

	if len(args) != 3 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	name := args[1]
	actionPipelineIDString := args[2]

	actionPipelineID, err := strconv.ParseInt(actionPipelineIDString, 10, 64)
	if err != nil {
		sdk.Exit("Error: actionPipelineID must be an integer\n")
	}

	err = sdk.DeletePipelineAction(name, projectKey, actionPipelineID)
	if err != nil {
		sdk.Exit("Error: cannot delete pipeline action %d (%s)\n", actionPipelineID, err)
	}

	fmt.Printf("Pipeline Action deleted.\n")
}
