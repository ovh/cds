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
	})
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
