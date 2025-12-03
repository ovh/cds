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
		cli.NewListCommand(projectV2ListCmd, projectV2ListRun, nil, withAllCommandModifiers()...),
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

var projectV2ListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS projects",
	Mcp:   true,
}

type CliProject struct {
	Name string `json:"name" cli:"name"`
	Key  string `json:"key" cli:"key"`
}

func projectV2ListRun(v cli.Values) (cli.ListResult, error) {
	projs, err := client.ProjectV2List(context.Background())
	if err != nil {
		return nil, err
	}
	cliProjects := make([]CliProject, 0, len(projs))
	for _, p := range projs {
		cliProj := CliProject{
			Name: p.Name,
			Key:  p.Key,
		}
		cliProjects = append(cliProjects, cliProj)
	}
	return cli.AsListResult(cliProjects), nil
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
