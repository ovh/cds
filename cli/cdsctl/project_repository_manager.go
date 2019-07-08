package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var projectRepositoryManagerCmd = cli.Command{
	Name:  "repository-manager",
	Short: "Manage CDS repository managers",
}

func projectRepositoryManager() *cobra.Command {
	return cli.NewCommand(projectRepositoryManagerCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectRepositoryManagerListCmd, projectRepositoryManagerListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(projectRepositoryManagerDeleteCmd, projectRepositoryManagerDeleteFunc, nil, withAllCommandModifiers()...),
	})
}

var projectRepositoryManagerListCmd = cli.Command{
	Name:  "list",
	Short: "List repository managers available on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectRepositoryManagerListFunc(v cli.Values) (cli.ListResult, error) {
	pfs, err := client.ProjectRepositoryManagerList(v.GetString(_ProjectKey))
	return cli.AsListResult(pfs), err
}

var projectRepositoryManagerDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a repository manager from a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectRepositoryManagerDeleteFunc(v cli.Values) error {
	return client.ProjectRepositoryManagerDelete(v.GetString(_ProjectKey), v.GetString("name"), v.GetBool("force"))
}
