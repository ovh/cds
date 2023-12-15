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
	})
}

var workflowRunStartJobCmd = cli.Command{
	Name:    "run",
	Aliases: []string{"start"},
	Short:   "Start a job",
	Example: "cdsctl workflow run <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number> <job_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run_number"},
		{Name: "job_name"},
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
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	runNumber, err := v.GetInt64("run_number")
	if err != nil {
		return err
	}
	job := v.GetString("job_name")
	data := v.GetString("data")

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return fmt.Errorf("unable to read json data")
	}

	run, err := client.WorkflowV2JobStart(context.Background(), projKey, vcsId, repoId, wkfName, runNumber, job, payload)
	if err != nil {
		return err
	}
	fmt.Printf("Workflow %s #%d.%d started", run.WorkflowName, run.RunNumber, run.RunAttempt)
	return nil
}

var workflowRunJobsCmd = cli.Command{
	Name:    "status",
	Short:   "Get the workflow run jobs status",
	Example: "cdsctl experimental workflow run jobs status <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run_number"},
	},
}

func workflowRunJobsFunc(v cli.Values) (cli.ListResult, error) {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	runNumber, err := v.GetInt64("run_number")
	if err != nil {
		return nil, err
	}

	runJobs, err := client.WorkflowV2RunJobs(context.Background(), projKey, vcsId, repoId, wkfName, runNumber)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(runJobs), nil
}

var workflowRunJobCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "Get the workflow run job status",
	Example: "cdsctl experimental workflow run jobs status <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number> <job_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run_number"},
		{Name: "job_identifier"},
	},
}

func workflowRunJobFunc(v cli.Values) (interface{}, error) {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	jobIdentifier := v.GetString("job_identifier")
	runNumber, err := v.GetInt64("run_number")
	if err != nil {
		return nil, err
	}

	runJob, err := client.WorkflowV2RunJob(context.Background(), projKey, vcsId, repoId, wkfName, jobIdentifier, runNumber)
	if err != nil {
		return nil, err
	}
	return runJob, nil
}

var workflowRunStopJobCmd = cli.Command{
	Name:    "stop",
	Short:   "Stop the workflow run job",
	Example: "cdsctl experimental workflow job stop <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number> <job_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run_number"},
		{Name: "job_name"},
	},
}

func workflowRunStopJobFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	runNumber, err := v.GetInt64("run_number")
	jobName := v.GetString("job_name")
	if err != nil {
		return err
	}

	if err := client.WorkflowV2StopJob(context.Background(), projKey, vcsId, repoId, wkfName, runNumber, jobName); err != nil {
		return err
	}
	fmt.Printf("Workflow run %d job %s has been stopped\n", runNumber, jobName)
	return nil
}
