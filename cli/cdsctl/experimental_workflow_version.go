package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalWorkflowVersionCmd = cli.Command{
	Name:    "version",
	Short:   "CDS Experimental workflow version commands",
	Aliases: []string{"versions"},
}

func experimentalWorkflowVersion() *cobra.Command {
	return cli.NewCommand(experimentalWorkflowVersionCmd, nil, []*cobra.Command{
		cli.NewListCommand(workflowV2VersionListCmd, workflowV2versionListFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(workflowV2VersionCmd, workflowV2VersionFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowV2VersionDeleteCmd, workflowV2VersionDeleteFunc, nil, withAllCommandModifiers()...),
	})
}

var workflowV2VersionDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete the workflow version",
	Example: "cdsctl experimental workflow version delete <project_key> <vcs_identifier> <repository_identifier> <workflow_name> <version>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repository_identifier"},
		{Name: "workflow_name"},
		{Name: "version"},
	},
}

func workflowV2VersionDeleteFunc(v cli.Values) error {
	if err := client.WorkflowV2VersionDelete(context.Background(), v.GetString("proj_key"), v.GetString("vcs_identifier"), v.GetString("repository_identifier"), v.GetString("workflow_name"), v.GetString("version")); err != nil {
		return err
	}
	return nil
}

var workflowV2VersionCmd = cli.Command{
	Name:    "get",
	Aliases: []string{"show"},
	Short:   "Get the workflow version",
	Example: "cdsctl experimental workflow version get <project_key> <vcs_identifier> <repository_identifier> <workflow_name> <version>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repository_identifier"},
		{Name: "workflow_name"},
		{Name: "version"},
	},
}

func workflowV2VersionFunc(v cli.Values) (interface{}, error) {
	workflowVersion, err := client.WorkflowV2VersionGet(context.Background(), v.GetString("proj_key"), v.GetString("vcs_identifier"), v.GetString("repository_identifier"), v.GetString("workflow_name"), v.GetString("version"))
	if err != nil {
		return nil, err
	}
	return workflowVersion, nil
}

var workflowV2VersionListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List all version for the given workflow",
	Example: "cdsctl experimental workflow version list <project_key> <vcs_identifier> <repository_identifier> <workflow_name>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repository_identifier"},
		{Name: "workflow_name"},
	},
}

func workflowV2versionListFunc(v cli.Values) (cli.ListResult, error) {
	versions, err := client.WorkflowV2VersionList(context.Background(), v.GetString("proj_key"), v.GetString("vcs_identifier"), v.GetString("repository_identifier"), v.GetString("workflow_name"))
	if err != nil {
		return nil, err
	}

	type cliVersion struct {
		ID       string    `cli:"id"`
		Version  string    `cli:"version"`
		Type     string    `cli:"type"`
		File     string    `cli:"file"`
		RunID    string    `json:"workflow_run_id" db:"workflow_run_id"`
		Username string    `json:"username" db:"username"`
		Created  time.Time `json:"created" db:"created"`
	}
	cliVersions := make([]cliVersion, 0, len(versions))
	for _, v := range versions {
		cliVersions = append(cliVersions, cliVersion{
			ID:       v.ID,
			Version:  v.Version,
			Type:     v.Type,
			File:     v.File,
			RunID:    v.WorkflowRunID,
			Username: v.Username,
			Created:  v.Created,
		})
	}

	return cli.AsListResult(cliVersions), nil
}
