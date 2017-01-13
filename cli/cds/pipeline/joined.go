package pipeline

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func pipelineJoinedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "joined",
		Short: "",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(pipelineJoinedActionCmd())
	return cmd
}

var cmdJoinedActionAddParams []string
var cmdJoinedActionAddStageNumber string

func pipelineJoinedActionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action",
		Short: "cds pipeline joined action {add | append | remove}",
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "cds pipeline joined action add <projectKey> <pipelineName> <joinedActionName> [-p <paramName>]",
		Long:  ``,
		Run:   pipelineJoinedActionAdd,
	}
	addCmd.Flags().StringSliceVarP(&cmdJoinedActionAddParams, "parameter", "p", nil, "Action parameters")
	addCmd.Flags().StringVarP(&cmdJoinedActionAddStageNumber, "stage", "", "0", "Stage number")

	appendCmd := &cobra.Command{
		Use:   "append",
		Short: "cds pipeline joined action append <projectKey> <pipelineName> <joinedActinName> <actionName> [-p <paramName>]",
		Long:  ``,
		Run:   pipelineJoinedActionAppend,
	}
	appendCmd.Flags().StringSliceVarP(&cmdJoinedActionAddParams, "parameter", "p", nil, "Action parameters")

	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "cds pipeline joined action remove <projectKey> <pipelineName> <joinedActionName>",
		Long:  ``,
		Run:   pipelineJoinedActionRemove,
	}

	cmd.AddCommand(addCmd)
	cmd.AddCommand(appendCmd)
	cmd.AddCommand(removeCmd)
	return cmd
}

func pipelineJoinedActionAdd(cmd *cobra.Command, args []string) {

	if len(args) != 3 {
		sdk.Exit("Wrong usage. See %s\n", cmd.Short)
	}

	projectKey := args[0]
	pipelineName := args[1]
	joinedActionName := args[2]

	a := sdk.NewAction(joinedActionName)

	for _, p := range cmdJoinedActionAddParams {
		a.Parameter(sdk.Parameter{Name: p, Type: sdk.StringParameter})
	}

	stage, err := strconv.ParseInt(cmdJoinedActionAddStageNumber, 10, 32)
	if err != nil {
		sdk.Exit("Error: stage is not a number (%s)\n", err)
	}

	// Get pipeline to get stage IDs
	var pipelineStageID int64
	pipeline, err := sdk.GetPipeline(projectKey, pipelineName)
	if err != nil {
		sdk.Exit("Error: cannot retrieve pipeline %s (%s)\n", pipelineName, err)
	}
	for i := range pipeline.Stages {
		if i+1 == int(stage) {
			pipelineStageID = pipeline.Stages[i].ID
		}
	}

	err = sdk.AddJoinedAction(projectKey, pipelineName, pipelineStageID, a)
	if err != nil {
		sdk.Exit("Error: cannot create joined action %s (%s)\n", joinedActionName, err)
	}
}

func pipelineJoinedActionAppend(cmd *cobra.Command, args []string) {

	if len(args) != 4 {
		sdk.Exit("Wrong usage. See %s\n", cmd.Short)
	}

	projectKey := args[0]
	pipelineName := args[1]
	joinedActionName := args[2]
	actionName := args[3]

	p, err := sdk.GetPipeline(projectKey, pipelineName)
	if err != nil {
		sdk.Exit("Error: cannot retrieve pipeline %s/%s (%s)", projectKey, pipelineName, err)
	}

	child, err := sdk.GetAction(actionName)
	if err != nil {
		sdk.Exit("Error: Cannot retrieve action %s (%s)\n", actionName, err)
	}

	// Find joined action
	var joinedAction sdk.Action
	var stage int
	for _, s := range p.Stages {
		for _, j := range s.Jobs {
			if j.Action.Name == joinedActionName {
				joinedAction = j.Action
				stage = s.BuildOrder
				break
			}
		}
		if joinedAction.Name != "" {
			break
		}
	}
	if joinedAction.Name == "" {
		sdk.Exit("Error: joined action %s not found in %s/%s\n", joinedActionName, projectKey, pipelineName)
	}

	for _, p := range cmdJoinedActionAddParams {
		t := strings.SplitN(p, "=", 2)
		if len(t) != 2 {
			sdk.Exit("Error: invalid parameter format (%s)", p)
		}
		found := false
		for i := range child.Parameters {
			if t[0] == child.Parameters[i].Name {
				found = true
				child.Parameters[i].Value = t[1]
				break
			}
		}
		if !found {
			sdk.Exit("Error: Argument %s does not exists in action %s\n", t[0], child.Name)
		}
	}

	joinedAction.Actions = append(joinedAction.Actions, child)

	err = sdk.UpdateJoinedAction(projectKey, pipelineName, stage, joinedAction)
	if err != nil {
		sdk.Exit("Error: cannot update joined action (%s)\n", err)
	}
}

func pipelineJoinedActionRemove(cmd *cobra.Command, args []string) {

	if len(args) != 3 {
		sdk.Exit("Wrong usage. See %s\n", cmd.Short)
	}

	sdk.Exit("Not implemented")
}
