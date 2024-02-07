package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalWorkflowJobCmd = cli.Command{
	Name:    "jobs",
	Aliases: []string{"job"},
	Short:   "CDS Experimental workflow job commands",
}

func experimentalWorkflowJob() *cobra.Command {
	return cli.NewCommand(experimentalWorkflowJobCmd, nil, []*cobra.Command{
		cli.NewListCommand(workflowRunJobsCmd, workflowRunJobsFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowRunStopJobCmd, workflowRunStopJobFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(workflowRunJobCmd, workflowRunJobFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowRunStartJobCmd, workflowRunStartJobFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(workflowRunJobInfoCmd, workflowRunJobInfoFunc, nil, withAllCommandModifiers()...),
	})
}

var workflowRunStartJobCmd = cli.Command{
	Name:    "run",
	Aliases: []string{"start"},
	Short:   "Start a job",
	Example: "cdsctl workflow run <proj_key> <run_identifier> <job_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
		{Name: "job_identifier"},
	},
	Flags: []cli.Flag{
		{
			Name:    "data",
			Default: "{}",
		},
	},
}

func workflowRunStartJobFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	jobIdentifier := v.GetString("job_identifier")
	data := v.GetString("data")
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return fmt.Errorf("unable to read json data")
	}
	run, err := client.WorkflowV2JobStart(context.Background(), projKey, runIdentifier, jobIdentifier, payload)
	if err != nil {
		return err
	}
	fmt.Printf("Workflow %s #%d.%d started", run.WorkflowName, run.RunNumber, run.RunAttempt)
	return nil
}

var workflowRunJobsCmd = cli.Command{
	Name:    "status",
	Short:   "Get the workflow run jobs status",
	Example: "cdsctl experimental workflow run jobs status <proj_key> <run_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
	},
}

func workflowRunJobsFunc(v cli.Values) (cli.ListResult, error) {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	runJobs, err := client.WorkflowV2RunJobs(context.Background(), projKey, runIdentifier)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(runJobs), nil
}

var workflowRunJobCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "Get the workflow run job status",
	Example: "cdsctl experimental workflow run jobs status <proj_key> <run_identifier> <job_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
		{Name: "job_identifier"},
	},
}

func workflowRunJobFunc(v cli.Values) (interface{}, error) {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	jobIdentifier := v.GetString("job_identifier")
	runJob, err := client.WorkflowV2RunJob(context.Background(), projKey, runIdentifier, jobIdentifier)
	if err != nil {
		return nil, err
	}
	return runJob, nil
}

var workflowRunJobInfoCmd = cli.Command{
	Name:    "info",
	Aliases: []string{"i", "infos"},
	Short:   "Get the workflow run job infos",
	Example: "cdsctl experimental workflow run jobs info <proj_key> <run_identifier> <job_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
		{Name: "job_identifier"},
	},
}

func workflowRunJobInfoFunc(v cli.Values) (cli.ListResult, error) {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	jobIdentifier := v.GetString("job_identifier")
	runJobInfoList, err := client.WorkflowV2RunJobInfoList(context.Background(), projKey, runIdentifier, jobIdentifier)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(runJobInfoList), nil
}

var workflowRunStopJobCmd = cli.Command{
	Name:    "stop",
	Short:   "Stop the workflow run job",
	Example: "cdsctl experimental workflow job stop <proj_key> <run_identifier> <job_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
		{Name: "job_identifier"},
	},
}

func workflowRunStopJobFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	jobIdentifier := v.GetString("job_identifier")
	if err := client.WorkflowV2StopJob(context.Background(), projKey, runIdentifier, jobIdentifier); err != nil {
		return err
	}
	fmt.Printf("Workflow run %s job %s has been stopped\n", runIdentifier, jobIdentifier)
	return nil
}
