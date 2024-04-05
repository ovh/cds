package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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
		cli.NewGetCommand(workflowRunCmd, workflowRunFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowRestartCmd, workflowRestartFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(workflowRunHistoryCmd, workflowRunHistoryFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(workflowRunInfosListCmd, workflowRunInfosListFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(workflowRunStatusCmd, workflowRunStatusFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowRunStopCmd, workflowRunStopFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowLintCmd, workflowLintFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(workflowRunSearchCmd, workflowRunSearchFunc, nil, withAllCommandModifiers()...),
		experimentalWorkflowRunLogs(),
		experimentalWorkflowJob(),
		experimentalWorkflowResult(),
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
	Name:    "info",
	Aliases: []string{"i", "infos"},
	Short:   "List run informations",
	Example: "cdsctl experimental workflow info <proj_key> <run_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
	},
}

func workflowRunInfosListFunc(v cli.Values) (cli.ListResult, error) {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	runInfos, err := client.WorkflowV2RunInfoList(context.Background(), projKey, runIdentifier)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(runInfos), nil
}

var workflowRunSearchCmd = cli.Command{
	Name:    "search",
	Aliases: []string{""},
	Short:   "Search a workflow run inside a project",
	Example: "cdsctl experimental workflow search",
	Ctx:     []cli.Arg{},
	Args:    []cli.Arg{},
	Flags: []cli.Flag{
		{Name: "commit"},
		{Name: "workflow", Usage: "<vcs>/<repo>/<workflow_name>"},
		{Name: "project"},
		{Name: "branch"},
		{Name: "status"},
		{Name: "actor"},
		{Name: "offset"},
		{Name: "limit"},
	},
}

func workflowRunSearchFunc(v cli.Values) (cli.ListResult, error) {
	commit := v.GetString("commit")
	workflow := v.GetString("workflow")
	projKey := v.GetString("project")
	branch := v.GetString("branch")
	status := v.GetString("status")
	actor := v.GetString("actor")

	offset, err := v.GetInt64("offset")
	if err != nil {
		return nil, err
	}
	limit, err := v.GetInt64("limit")
	if err != nil {
		return nil, err
	}
	if limit == 0 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	mods := make([]cdsclient.RequestModifier, 0)
	if commit != "" {
		mods = append(mods, cdsclient.WithQueryParameter("commit", commit))
	}
	if workflow != "" {
		mods = append(mods, cdsclient.Workflows(workflow))
	}
	if branch != "" {
		mods = append(mods, cdsclient.WithQueryParameter("branch", branch))
	}
	if actor != "" {
		mods = append(mods, cdsclient.WithQueryParameter("actor", actor))
	}
	if status != "" {
		mods = append(mods, cdsclient.WithQueryParameter("status", status))
	}

	var runs []sdk.V2WorkflowRun
	if projKey == "" {
		// Search accross all project
		runs, err = client.WorkflowV2RunSearchAllProjects(context.Background(), offset, limit, mods...)
	} else {
		mods = append(mods, cdsclient.WithQueryParameter("offset", fmt.Sprintf("%d", offset)))
		mods = append(mods, cdsclient.WithQueryParameter("limit", fmt.Sprintf("%d", limit)))
		runs, err = client.WorkflowV2RunSearch(context.Background(), projKey, mods...)
	}

	if err != nil {
		return nil, err
	}

	return cli.AsListResult(runs), nil
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
	Flags: []cli.Flag{
		{
			Name: "commit",
		},
	},
}

func workflowRunHistoryFunc(v cli.Values) (cli.ListResult, error) {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	commit := v.GetString("commit")

	wkfIdentifier := vcsId + "/" + repoId + "/" + wkfName

	mods := make([]cdsclient.RequestModifier, 0)
	mods = append(mods, cdsclient.Workflows(wkfIdentifier))

	if commit != "" {
		mods = append(mods, cdsclient.WithQueryParameter("commit", commit))
	}

	runs, err := client.WorkflowV2RunSearch(context.Background(), projKey, mods...)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(runs), nil
}

var workflowRunStatusCmd = cli.Command{
	Name:    "status",
	Aliases: []string{"st"},
	Short:   "Get the workflow run status",
	Example: "cdsctl experimental workflow status <proj_key> <run_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
	},
}

func workflowRunStatusFunc(v cli.Values) (interface{}, error) {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	run, err := client.WorkflowV2RunStatus(context.Background(), projKey, runIdentifier)
	if err != nil {
		return nil, err
	}
	return run, nil
}

var workflowRunStopCmd = cli.Command{
	Name:    "stop",
	Aliases: []string{""},
	Short:   "Stop the workflow run",
	Example: "cdsctl experimental workflow stop <proj_key> <run_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
	},
}

func workflowRunStopFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	if err := client.WorkflowV2Stop(context.Background(), projKey, runIdentifier); err != nil {
		return err
	}
	fmt.Printf("Workflow run %s has been stopped\n", runIdentifier)
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
			Name: "tag",
		},
		{
			Name: "commit",
		},
		{
			Name: "workflow-branch",
		},
		{
			Name: "workflow-tag",
		},
	},
}

func workflowRunFunc(v cli.Values) (interface{}, error) {
	projKey := v.GetString("proj_key")
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	destBranch := v.GetString("branch")
	destTag := v.GetString("tag")
	destCommit := v.GetString("commit")
	workflowBranch := v.GetString("workflow-branch")
	workflowTag := v.GetString("workflow-tag")

	if destBranch != "" && destTag != "" {
		return nil, fmt.Errorf("you cannot use branch and tag together")
	}

	if workflowBranch != "" && workflowTag != "" {
		return nil, fmt.Errorf("you cannot use workflow-branch and workflow-tag together")
	}

	payload := sdk.V2WorkflowRunManualRequest{
		Branch:         destBranch,
		Tag:            destTag,
		Sha:            destCommit,
		WorkflowBranch: workflowBranch,
		WorkflowTag:    workflowTag,
	}

	runResp, err := client.WorkflowV2Run(context.Background(), projKey, vcsId, repoId, wkfName, payload)
	if err != nil {
		return nil, err
	}

	type run struct {
		Workflow  string `json:"workflow" cli:"workflow"`
		RunNumber int64  `json:"run_number" cli:"run_number"`
		RunID     string `json:"run_id" cli:"run_id"`
		Error     string `json:"error" cli:"error"`
		UIUrl     string `json:"uri_url" cli:"ui_url"`
	}

	retry := 0
	for {
		event, err := client.ProjectRepositoryEvent(context.Background(), projKey, vcsId, repoId, runResp.HookEventUUID)
		if err != nil {
			return nil, err
		}
		if event.Status == sdk.HookEventStatusDone {
			if len(event.WorkflowHooks) == 1 {
				return run{
					Workflow:  wkfName,
					RunNumber: event.WorkflowHooks[0].RunNumber,
					RunID:     event.WorkflowHooks[0].RunID,
					UIUrl:     fmt.Sprintf("%s/projectv2/%s/run/%s", runResp.UIUrl, projKey, event.WorkflowHooks[0].RunID),
				}, nil
			}
			return nil, fmt.Errorf("workflow did not start")
		}
		if event.Status == sdk.HookEventStatusError || event.Status == sdk.HookEventStatusSkipped {
			return run{
				Error: event.LastError,
			}, nil
		}
		retry++
		if retry > 90 {
			return nil, fmt.Errorf("workflow take too much time to start")
		}
		time.Sleep(1 * time.Second)
	}
}

var workflowRestartCmd = cli.Command{
	Name:    "restart",
	Short:   "Restart workflow failed jobs",
	Example: "cdsctl workflow restart <proj_key> <run_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
	},
}

func workflowRestartFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	run, err := client.WorkflowV2Restart(context.Background(), projKey, runIdentifier)
	if err != nil {
		return err
	}
	fmt.Printf("Worflow %s #%d.%d restarted", run.WorkflowName, run.RunNumber, run.RunAttempt)
	return nil
}
