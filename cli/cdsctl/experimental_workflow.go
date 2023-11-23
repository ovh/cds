package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rockbears/yaml"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var experimentalWorkflowCmd = cli.Command{
	Name:  "workflow",
	Short: "CDS Experimental workflow commands",
}

func experimentalWorkflow() *cobra.Command {
	return cli.NewCommand(experimentalWorkflowCmd, nil, []*cobra.Command{
		cli.NewCommand(workflowRunCmd, workflowRunFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowReRunCmd, workflowReRunFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(workflowRunInfosListCmd, workflowRunInfosListFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(workflowRunHistoryCmd, workflowRunHistoryFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(workflowRunStatusCmd, workflowRunStatusFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowRunStopCmd, workflowRunStopFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowLintCmd, workflowLintFunc, nil, withAllCommandModifiers()...),
		experimentalWorkflowRunLogs(),
		experimentalWorkflowJob(),
	})
}

var workflowLintCmd = cli.Command{
	Name:    "lint",
	Short:   "Lint workflow files",
	Example: "cdsctl experimental workflow lint .cds/workflows",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "cds_workflow_directory"},
	},
}

func workflowLintFunc(v cli.Values) error {
	files, err := os.ReadDir(v.GetString("cds_workflow_directory"))
	if err != nil {
		return err
	}

	hasErrors := false
	for _, f := range files {
		bts, err := os.ReadFile(fmt.Sprintf("%s/%s", strings.TrimSuffix(v.GetString("cds_workflow_directory"), "/"), f.Name()))
		if err != nil {
			return err
		}
		var wf sdk.V2Workflow
		if err := yaml.Unmarshal(bts, &wf); err != nil {
			fmt.Printf("File %s: unable to unmarshal yaml: %v\n", f.Name(), err)
			hasErrors = true
			continue
		}
		resp, err := client.EntityLint(context.Background(), sdk.EntityTypeWorkflow, wf)
		if err != nil {
			fmt.Printf("File %s: unable to check file: %v\n", f.Name(), err)
			hasErrors = true
			continue
		}
		if len(resp.Messages) == 0 {
			fmt.Printf("File %s: workflow OK\n", f.Name())
			continue
		}
		fmt.Printf("File %s: %d errors found\n", f.Name(), len(resp.Messages))
		for _, e := range resp.Messages {
			fmt.Printf("    %s\n", e)
		}
		hasErrors = true
	}
	if hasErrors {
		cli.OSExit(1)
	}
	return nil
}

var workflowRunInfosListCmd = cli.Command{
	Name:    "infos",
	Aliases: []string{"i", "info"},
	Short:   "List run informations",
	Example: "cdsctl experimental workflow infos <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run_number"},
	},
}

func workflowRunInfosListFunc(v cli.Values) (cli.ListResult, error) {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	runNumber, err := v.GetInt64("run_number")
	if err != nil {
		return nil, err
	}

	runInfos, err := client.WorkflowV2RunInfoList(context.Background(), projKey, vcsId, repoId, wkfName, runNumber)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(runInfos), nil
}

var workflowRunHistoryCmd = cli.Command{
	Name:    "history",
	Aliases: []string{"h"},
	Short:   "Display the run history for the given workflow",
	Example: "cdsctl experimental workflow history <proj_key> <vcs_identifier> <repo_identifier> <workflow_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
	},
}

func workflowRunHistoryFunc(v cli.Values) (cli.ListResult, error) {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")

	runs, err := client.WorkflowV2RunList(context.Background(), projKey, vcsId, repoId, wkfName)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(runs), nil
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

var workflowRunStopCmd = cli.Command{
	Name:    "stop",
	Aliases: []string{""},
	Short:   "Stop the workflow run",
	Example: "cdsctl experimental workflow stop <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run-number"},
	},
}

func workflowRunStopFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	runNumber, err := v.GetInt64("run-number")
	if err != nil {
		return err
	}

	if err := client.WorkflowV2Stop(context.Background(), projKey, vcsId, repoId, wkfName, runNumber); err != nil {
		return err
	}
	fmt.Printf("Workflow run %d has been stopped\n", runNumber)
	return nil
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
		{
			Name:    "data",
			Default: "{}",
		},
	},
}

func workflowRunFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	branch := v.GetString("branch")
	data := v.GetString("data")

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return fmt.Errorf("unable to read json data")
	}

	run, err := client.WorkflowV2Run(context.Background(), projKey, vcsId, repoId, wkfName, payload, cdsclient.WithQueryParameter("branch", branch))
	if err != nil {
		return err
	}
	fmt.Printf("Worflow %s #%d started", run.WorkflowName, run.RunNumber)
	return nil
}

var workflowReRunCmd = cli.Command{
	Name:    "rerun",
	Short:   "Re run a new workflow",
	Example: "cdsctl workflow run <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run_number"},
	},
}

func workflowReRunFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	runNumber, err := v.GetInt64("run_number")
	if err != nil {
		return err
	}

	run, err := client.WorkflowV2ReRun(context.Background(), projKey, vcsId, repoId, wkfName, runNumber)
	if err != nil {
		return err
	}
	fmt.Printf("Worflow %s #%d.%d restarted", run.WorkflowName, run.RunNumber, run.RunAttempt)
	return nil
}
