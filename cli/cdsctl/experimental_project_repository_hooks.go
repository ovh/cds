package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectRepositorHooksCmd = cli.Command{
	Name:    "hooks",
	Aliases: []string{"hook"},
	Short:   "Manage hook on a repository",
}

func projectRepositoryHooks() *cobra.Command {
	return cli.NewCommand(projectRepositorHooksCmd, nil, []*cobra.Command{
		cli.NewGetCommand(projectRepositoryHookAddCmd, projectRepositoryHookAddFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(projectRepositoryHookListCmd, projectRepositoryHookListFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectRepositoryHookGetCmd, projectRepositoryHookGetFunc, nil, withAllCommandModifiers()...),

		cli.NewDeleteCommand(projectRepositoryDeleteCmd, projectRepositoryDeleteFunc, nil, withAllCommandModifiers()...),
	})
}

var projectRepositoryHookAddCmd = cli.Command{
	Name:    "add",
	Aliases: []string{"create"},
	Short:   "Ask for a hook secret",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository-name"},
	},
}

func projectRepositoryHookAddFunc(v cli.Values) (interface{}, error) {
	r := sdk.PostProjectRepositoryHook{
		VCSServer:  v.GetString("vcs-name"),
		Repository: v.GetString("repository-name"),
	}
	return client.ProjectRepositoryHookAdd(context.Background(), v.GetString(_ProjectKey), r)
}

var projectRepositoryHookListCmd = cli.Command{
	Name:  "list",
	Short: "List availablehooks on project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectRepositoryHookListFunc(v cli.Values) (cli.ListResult, error) {
	hooks, err := client.ProjectRepositoryHookList(context.Background(), v.GetString(_ProjectKey))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(hooks), nil
}

var projectRepositoryHookDeleteCmd = cli.Command{
	Name:    "delete",
	Short:   "Remove a repository hook from on a project",
	Aliases: []string{"remove", "rm"},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "hook-id"},
	},
}

func projectRepositoryHookDeleteFunc(v cli.Values) error {
	return client.ProjectRepositoryHookDelete(context.Background(), v.GetString(_ProjectKey), v.GetString("hook-id"))
}

var projectRepositoryHookGetCmd = cli.Command{
	Name:    "get",
	Short:   "Get a repository hook",
	Aliases: []string{"show"},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "hook-id"},
	},
}

func projectRepositoryHookGetFunc(v cli.Values) (interface{}, error) {
	return client.ProjectRepositoryHookGet(context.Background(), v.GetString(_ProjectKey), v.GetString("hook-id"))
}
