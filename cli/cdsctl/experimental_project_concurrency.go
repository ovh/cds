package main

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectConcurrencyCmd = cli.Command{
	Name:    "concurrency",
	Aliases: []string{"concurrencies"},
	Short:   "Manage Concurrency rule on a CDS project",
}

func projectConcurrency() *cobra.Command {
	return cli.NewCommand(projectConcurrencyCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectConcurrencyListCmd, projectConcurrencyListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(projectConcurrencyDeleteCmd, projectConcurrencyDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectConcurrencyCreateCmd, projectConcurrencyCreateFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectConcurrencyUpdateCmd, projectConcurrencyUpdateFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectConcurrencyShowCmd, projectConcurrencyShowFunc, nil, withAllCommandModifiers()...),
	})
}

var projectConcurrencyShowCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},

	Short: "Get the given concurrency rule",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectConcurrencyShowFunc(v cli.Values) (interface{}, error) {
	pc, err := client.ProjectConcurrencyGet(context.Background(), v.GetString(_ProjectKey), v.GetString("name"))
	return pc, err
}

var projectConcurrencyListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List all concurrency rules in the given project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectConcurrencyListFunc(v cli.Values) (cli.ListResult, error) {
	pcs, err := client.ProjectConcurrencyList(context.Background(), v.GetString(_ProjectKey))
	return cli.AsListResult(pcs), err
}

var projectConcurrencyDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a concurrency rule on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectConcurrencyDeleteFunc(v cli.Values) error {
	return client.ProjectConcurrencyDelete(context.Background(), v.GetString(_ProjectKey), v.GetString("name"))
}

var projectConcurrencyCreateCmd = cli.Command{
	Name:    "add",
	Aliases: []string{"create"},
	Short:   "Create a new concurrency inside the given project",
	Example: "cdsctl X project concurrency add MY-PROJECT MY-CONCURRENCY-NAME MY_DESCRIPTIOn --pool 1 --order oldest_first --cancel-in-progress",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
		{Name: "description"},
	},
	Flags: []cli.Flag{
		{Name: "pool", Type: cli.FlagString, Default: "1"},
		{Name: "order", Type: cli.FlagString, Default: string(sdk.ConcurrencyOrderOldestFirst)},
		{Name: "cancel-in-progress", Type: cli.FlagBool, Default: "false"},
	},
}

func projectConcurrencyCreateFunc(v cli.Values) error {
	pc := sdk.ProjectConcurrency{
		Name:             v.GetString("name"),
		Description:      v.GetString("description"),
		ProjectKey:       v.GetString(_ProjectKey),
		Order:            sdk.ConcurrencyOrder(v.GetString("order")),
		CancelInProgress: v.GetBool("cancel-in-progress"),
	}
	if v.GetString("pool") != "" {
		pool, err := strconv.ParseInt(v.GetString("pool"), 10, 64)
		if err != nil {
			return err
		}
		pc.Pool = pool
	}

	return client.ProjectConcurrencyCreate(context.Background(), v.GetString(_ProjectKey), &pc)
}

var projectConcurrencyUpdateCmd = cli.Command{
	Name:    "update",
	Aliases: []string{"up"},
	Short:   "Update a the given concurrency inside the given project",
	Example: "cdsctl X project concurrency update MY-PROJECT CONCURRENCY_NAME --description=<new description> --rename=<new_name> --pool 1 --order newest_first --cancel-in-progress",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
	Flags: []cli.Flag{
		{Name: "description", Type: cli.FlagString},
		{Name: "pool", Type: cli.FlagString, Default: "1"},
		{Name: "order", Type: cli.FlagString, Default: string(sdk.ConcurrencyOrderOldestFirst)},
		{Name: "cancel-in-progress", Type: cli.FlagBool, Default: "false"},
	},
}

func projectConcurrencyUpdateFunc(v cli.Values) error {
	pc, err := client.ProjectConcurrencyGet(context.Background(), v.GetString(_ProjectKey), v.GetString("name"))
	if err != nil {
		return err
	}
	if v.GetString("description") != "" {
		pc.Description = v.GetString("description")
	}
	if v.GetString("order") != "" {
		pc.Order = sdk.ConcurrencyOrder(v.GetString("order"))
	}
	if v.GetString("pool") != "" {
		pool, err := strconv.ParseInt(v.GetString("pool"), 10, 64)
		if err != nil {
			return err
		}
		pc.Pool = pool
	}
	if v.GetBool("cancel-in-progress") != pc.CancelInProgress {
		pc.CancelInProgress = v.GetBool("cancel-in-progress")
	}
	return client.ProjectConcurrencyUpdate(context.Background(), v.GetString(_ProjectKey), pc)
}
