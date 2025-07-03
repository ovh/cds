package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalProjectCmd = cli.Command{
	Name:  "project",
	Short: "CDS Experimental project commands",
}

func experimentalProject() *cobra.Command {
	return cli.NewCommand(experimentalProjectCmd, nil, []*cobra.Command{
		cli.NewCommand(projectDeleteRunCmd, projectDeleteRunCmdFunc, nil, withAllCommandModifiers()...),
		projectRepository(),
		projectRepositoryAnalysis(),
		projectNotification(),
		projectVariableSet(),
		projectConcurrency(),
		projectWebHooks(),
		projectRetention(),
	})
}

var projectDeleteRunCmd = cli.Command{
	Name:    "delete-runs",
	Aliases: []string{},
	Short:   "Delete workflow runs regarding project retention rules",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{},
}

func projectDeleteRunCmdFunc(v cli.Values) error {
	if err := client.ProjectRunPurge(context.Background(), v.GetString(_ProjectKey)); err != nil {
		return err
	}
	return nil
}
