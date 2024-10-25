package main

import (
	"context"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectRepositoryCmd = cli.Command{
	Name:    "repository",
	Aliases: []string{"repo"},
	Short:   "Manage repositories on a CDS project",
}

func projectRepository() *cobra.Command {
	return cli.NewCommand(projectRepositoryCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectRepositoryListCmd, projectRepositoryListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(projectRepositoryDeleteCmd, projectRepositoryDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectRepositoryAddCmd, projectRepositoryAddFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectRepositoryHookSecretCmd, projectRepositoryHookSecretFunc, nil, withAllCommandModifiers()...),
	})
}

var projectRepositoryAddCmd = cli.Command{
	Name:  "add",
	Short: "Add a repository on the project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository-name"},
	},
}

func projectRepositoryAddFunc(v cli.Values) error {
	repo := sdk.ProjectRepository{
		Name: v.GetString("repository-name"),
	}
	return client.ProjectVCSRepositoryAdd(context.Background(), v.GetString(_ProjectKey), v.GetString("vcs-name"), repo)
}

var projectRepositoryListCmd = cli.Command{
	Name:  "list",
	Short: "List available repositories on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Flags: []cli.Flag{
		{Name: "vcs-name", Usage: "Specified a VCS by its name"},
	},
}

func projectRepositoryListFunc(v cli.Values) (cli.ListResult, error) {
	// GET VCS
	vcsGivenName := v.GetString("vcs-name")
	allVCS := make([]string, 0)
	if vcsGivenName == "" {
		vcs, err := client.ProjectVCSList(context.Background(), v.GetString(_ProjectKey))
		if err != nil {
			return nil, err
		}
		for _, v := range vcs {
			allVCS = append(allVCS, v.Name)
		}
	} else {
		allVCS = append(allVCS, vcsGivenName)
	}

	type CliRepo struct {
		ID       string `cli:"id" json:"id"`
		VcsName  string `cli:"vcsName" json:"vcsName"`
		RepoName string `cli:"repoName" json:"repoName"`
	}

	// GET REPOS
	repositories := make([]CliRepo, 0)
	for _, vcsName := range allVCS {
		repos, err := client.ProjectVCSRepositoryList(context.Background(), v.GetString(_ProjectKey), vcsName)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			repositories = append(repositories, CliRepo{
				VcsName:  vcsName,
				RepoName: r.Name,
				ID:       r.ID,
			})
		}
	}
	return cli.AsListResult(repositories), nil
}

var projectRepositoryDeleteCmd = cli.Command{
	Name:    "delete",
	Short:   "Remove a repository from on a project",
	Aliases: []string{"remove", "rm"},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository-name"},
	},
}

func projectRepositoryDeleteFunc(v cli.Values) error {
	return client.ProjectRepositoryDelete(context.Background(), v.GetString(_ProjectKey), v.GetString("vcs-name"), v.GetString("repository-name"))
}

var projectRepositoryHookSecretCmd = cli.Command{
	Name:    "hook-key",
	Short:   "Get repository webhook secret key",
	Aliases: []string{"hk"},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-type"},
		{Name: "vcs-name"},
		{Name: "repository-name"},
	},
}

func projectRepositoryHookSecretFunc(v cli.Values) (interface{}, error) {
	return client.ProjectRepositoryHookSecret(context.Background(), v.GetString(_ProjectKey), v.GetString("vcs-type"), v.GetString("vcs-name"), v.GetString("repository-name"))
}
