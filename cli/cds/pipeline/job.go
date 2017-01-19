package pipeline

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func pipelineJoinedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(pipelineJobCmd())
	return cmd
}

var cmdJoinedActionAddParams []string
var cmdJoinedActionAddStageNumber string

func pipelineJobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "cds pipeline job {add | append | remove}",
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "cds pipeline job add <projectKey> <pipelineName> <jobName>",
		Long:  ``,
		Run:   pipelineJobAdd,
	}
	addCmd.Flags().StringVarP(&cmdJoinedActionAddStageNumber, "stage", "", "0", "Stage number")

	appendCmd := &cobra.Command{
		Use:   "append",
		Short: "cds pipeline job append <projectKey> <pipelineName> <jobName> <actionName> [-p <paramName>]",
		Long:  ``,
		Run:   pipelineJobAppend,
	}
	appendCmd.Flags().StringSliceVarP(&cmdJoinedActionAddParams, "parameter", "p", nil, "Action parameters")

	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "cds pipeline job remove <projectKey> <pipelineName> <jobName>",
		Long:  ``,
		Run:   pipelineJobRemove,
	}

	cmd.AddCommand(addCmd)
	cmd.AddCommand(appendCmd)
	cmd.AddCommand(removeCmd)
	return cmd
}

func pipelineJobAdd(cmd *cobra.Command, args []string) {

	if len(args) != 3 {
		sdk.Exit("Wrong usage. See %s\n", cmd.Short)
	}

	projectKey := args[0]
	pipelineName := args[1]
	jobName := args[2]

	a := sdk.NewAction(jobName)

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

	job := &sdk.Job {
		PipelineStageID: pipelineStageID,
		Enabled: true,
		Action: *a,
	}

	err = sdk.AddJob(projectKey, pipelineName, job)
	if err != nil {
		sdk.Exit("Error: cannot create joined action %s (%s)\n", jobName, err)
	}
}

func pipelineJobAppend(cmd *cobra.Command, args []string) {

	if len(args) != 4 {
		sdk.Exit("Wrong usage. See %s\n", cmd.Short)
	}

	projectKey := args[0]
	pipelineName := args[1]
	jobName := args[2]
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
	var job *sdk.Job
	var stage int
	for _, s := range p.Stages {
		for _, j := range s.Jobs {
			if j.Action.Name == jobName {
				job = &j
				stage = s.BuildOrder
				break
			}
		}
		if job.Action.Name != "" {
			sdk.Exit("Error: job %s not found in %s/%s\n", jobName, projectKey, pipelineName)
		}
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

	job.Action.Actions = append(job.Action.Actions, child)

	err = sdk.UpdateJoinedAction(projectKey, pipelineName, stage, job)
	if err != nil {
		sdk.Exit("Error: cannot update joined action (%s)\n", err)
	}
}

func pipelineJobRemove(cmd *cobra.Command, args []string) {

	if len(args) != 3 {
		sdk.Exit("Wrong usage. See %s\n", cmd.Short)
	}

	sdk.Exit("Not implemented")
}
