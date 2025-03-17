package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectWebHooksCmd = cli.Command{
	Name:    "hooks",
	Aliases: []string{"hook"},
	Short:   "Manage webhook on a project",
}

func projectWebHooks() *cobra.Command {
	return cli.NewCommand(projectWebHooksCmd, nil, []*cobra.Command{
		cli.NewGetCommand(projectWebHookAddCmd, projectWebHookAddFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(projectWebHookListCmd, projectWebHookListFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectWebHookGetCmd, projectWebHookGetFunc, nil, withAllCommandModifiers()...),

		cli.NewDeleteCommand(projectWebHookDeleteCmd, projectWebHookDeleteFunc, nil, withAllCommandModifiers()...),
	})
}

var projectWebHookAddCmd = cli.Command{
	Name:    "add",
	Aliases: []string{"create"},
	Short:   "Create a project webhook",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository-name"},
	},
	Flags: []cli.Flag{
		{Name: "workflow", Type: cli.FlagString, Default: ""},
	},
}

func projectWebHookAddFunc(v cli.Values) (interface{}, error) {
	r := sdk.PostProjectWebHook{
		VCSServer:  v.GetString("vcs-name"),
		Repository: v.GetString("repository-name"),
		Workflow:   v.GetString("workflow"),
	}
	if r.Workflow == "" {
		r.Type = sdk.ProjectWebHookTypeRepository
	} else {
		r.Type = sdk.ProjectWebHookTypeWorkflow
	}
	return client.ProjectWebHookAdd(context.Background(), v.GetString(_ProjectKey), r)
}

var projectWebHookListCmd = cli.Command{
	Name:  "list",
	Short: "List availablehooks on project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectWebHookListFunc(v cli.Values) (cli.ListResult, error) {
	hooks, err := client.ProjectWebHookList(context.Background(), v.GetString(_ProjectKey))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(hooks), nil
}

var projectWebHookDeleteCmd = cli.Command{
	Name:    "delete",
	Short:   "Remove a  webhook from on a project",
	Aliases: []string{"remove", "rm"},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "hook-id"},
	},
}

func projectWebHookDeleteFunc(v cli.Values) error {
	return client.ProjectWebHookDelete(context.Background(), v.GetString(_ProjectKey), v.GetString("hook-id"))
}

var projectWebHookGetCmd = cli.Command{
	Name:    "get",
	Short:   "Get a webhook",
	Aliases: []string{"show"},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "hook-id"},
	},
}

func projectWebHookGetFunc(v cli.Values) (interface{}, error) {
	return client.ProjectWebHookGet(context.Background(), v.GetString(_ProjectKey), v.GetString("hook-id"))
}
