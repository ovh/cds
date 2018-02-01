package main

import (
	"fmt"

	repo "github.com/fsamin/go-repo"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	workflowCmd = cli.Command{
		Name:  "workflow",
		Short: "Manage CDS workflow",
	}

	workflow = cli.NewCommand(workflowCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(workflowListCmd, workflowListRun, nil, withAllCommandModifiers()...),
			cli.NewListCommand(workflowHistoryCmd, workflowHistoryRun, nil, withAllCommandModifiers()...),
			cli.NewGetCommand(workflowShowCmd, workflowShowRun, nil, withAllCommandModifiers()...),
			cli.NewGetCommand(workflowStatusCmd, workflowStatusRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(workflowRunManualCmd, workflowRunManualRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(workflowStopCmd, workflowStopRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(workflowExportCmd, workflowExportRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(workflowImportCmd, workflowImportRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(workflowPullCmd, workflowPullRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(workflowPushCmd, workflowPushRun, nil, withAllCommandModifiers()...),
			workflowArtifact,
			workflowAdvanced,
		})
)

func workflowNodeForCurrentRepo(projectKey, workflowName string) (int64, error) {
	//Try to get the latest commit
	r, err := repo.New(".")
	if err != nil {
		return 0, nil
	}

	latestCommit, err := r.LatestCommit()
	if err != nil {
		return 0, fmt.Errorf("unable to get latest commit: %v", err)
	}

	filters := []cdsclient.Filter{
		{
			Name:  "workflow",
			Value: workflowName,
		},
		{
			Name:  "git.hash",
			Value: latestCommit.Hash,
		},
	}
	//Searching workflow
	runs, err := client.WorkflowRunSearch(projectKey, 0, 0, filters...)
	if err != nil {
		return 0, err
	}
	if len(runs) != 1 {
		return 0, fmt.Errorf("workflow run not found : %+v", runs)
	}

	if runs[0].Number > 0 {
		return runs[0].Number, nil
	}

	// If no run number, get the latest
	runs, err = client.WorkflowRunList(projectKey, workflowName, 0, 1)
	if err != nil {
		return 0, err
	}

	if len(runs) != 1 {
		return 0, fmt.Errorf("workflow run not found : %+v", runs)
	}

	return runs[0].Number, nil
}
