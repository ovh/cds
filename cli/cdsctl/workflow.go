package main

import (
	"context"

	repo "github.com/fsamin/go-repo"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

var workflowCmd = cli.Command{
	Name:    "workflow",
	Aliases: []string{"workflows"},
	Short:   "Manage CDS workflow",
}

func workflow() *cobra.Command {
	return cli.NewCommand(workflowCmd, nil, []*cobra.Command{
		cli.NewCommand(workflowInitCmd, workflowInitRun, nil),
		cli.NewCommand(templateApplyCmd("applyTemplate"), templateApplyRun, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(workflowRunDeleteCmd, workflowRunDelete, nil, withAllCommandModifiers()...),
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
		cli.NewCommand(workflowFavoriteCmd, workflowFavoriteRun, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(workflowTransformAsCodeCmd, workflowTransformAsCodeRun, nil, withAllCommandModifiers()...),
		workflowLabel(),
		workflowArtifact(),
		workflowLog(),
		workflowAdvanced(),
		workflowRunResult(),
	})
}

func workflowNodeForCurrentRepo(projectKey, workflowName string) (int64, error) {
	//Try to get the latest commit
	ctx := context.Background()
	r, err := repo.New(ctx, "")
	if err != nil {
		return 0, nil
	}

	latestCommit, err := r.LatestCommit(ctx, repo.CommitOption{DisableDiffDetail: true})
	if err != nil {
		return 0, cli.WrapError(err, "unable to get latest commit")
	}

	filters := []cdsclient.Filter{
		{
			Name:  "workflow",
			Value: workflowName,
		},
		{
			Name:  "git.hash",
			Value: latestCommit.LongHash,
		},
	}

	//Searching workflow
	runs, err := client.WorkflowRunSearch(projectKey, 0, 0, filters...)
	if err != nil {
		return 0, err
	}
	if len(runs) < 1 {
		return 0, cli.NewError("workflow run not found")
	}

	if runs[0].Number > 0 {
		return runs[0].Number, nil
	}

	return 0, nil
}
