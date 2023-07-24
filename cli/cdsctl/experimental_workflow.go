package main

import (
	"context"
	"fmt"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalWorkflowCmd = cli.Command{
	Name:  "workflow",
	Short: "CDS Experimental workflow commands",
}

func experimentalWorkflow() *cobra.Command {
	return cli.NewCommand(experimentalWorkflowCmd, nil, []*cobra.Command{
		cli.NewCommand(workflowRunCmd, workflowRunFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(workflowRunStatusCmd, workflowRunStatusFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(workflowRunJobsCmd, workflowRunJobsFunc, nil, withAllCommandModifiers()...),
		experimentalWorkflowRunJobs(),
	})
}

var workflowRunStatusCmd = cli.Command{
	Name:    "status",
	Aliases: []string{"st"},
	Short:   "Get the workflow run status",
	Example: "cdsctl experimental workflow status <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run-number"},
	},
}

func workflowRunStatusFunc(v cli.Values) (interface{}, error) {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	runNumber, err := v.GetInt64("run-number")
	if err != nil {
		return nil, err
	}

	run, err := client.WorkflowV2RunStatus(context.Background(), projKey, vcsId, repoId, wkfName, runNumber)
	if err != nil {
		return nil, err
	}
	return run, nil
}

var workflowRunCmd = cli.Command{
	Name:    "run",
	Aliases: []string{"start"},
	Short:   "Start a new workflow",
	Example: "cdsctl workflow run <proj_key> <vcs_identifier> <repo_identifier> <workflow_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
	},
	Flags: []cli.Flag{
		{
			Name: "branch",
		},
	},
}

func workflowRunFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	branch := v.GetString("branch")

	run, err := client.WorkflowV2Run(context.Background(), projKey, vcsId, repoId, wkfName, cdsclient.WithQueryParameter("branch", branch))
	if err != nil {
		return err
	}
	fmt.Printf("Worflow %s #%d started", run.WorkflowName, run.RunNumber)
	return nil
}

var workflowRunJobsCmd = cli.Command{
	Name:    "jobs",
	Aliases: []string{"job"},
	Short:   "Get the workflow run jobs status",
	Example: "cdsctl experimental workflow run jobs status <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run-number"},
	},
}

func workflowRunJobsFunc(v cli.Values) (cli.ListResult, error) {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	runNumber, err := v.GetInt64("run-number")
	if err != nil {
		return nil, err
	}

	runJobs, err := client.WorkflowV2RunJobs(context.Background(), projKey, vcsId, repoId, wkfName, runNumber)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(runJobs), nil
}
