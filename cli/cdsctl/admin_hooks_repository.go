package main

import (
	"fmt"
	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
	"net/url"
)

var adminHooksRepositoryCmd = cli.Command{
	Name:    "repository",
	Aliases: []string{"r", "repo", "repositories"},
	Short:   "Manage repositories where there were events",
}

func adminHooksRepositories() *cobra.Command {
	return cli.NewCommand(adminHooksRepositoryCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminHooksRepoListCmd, adminHooksRepoListRun, nil),
		cli.NewDeleteCommand(adminHookRepoDeleteCmd, adminHookRepoDeleteRun, nil),
		adminHooksRepositoryEvents(),
	})
}

var adminHookRepoDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"rm", "remove"},
	Short:   "Remove repository",
	Args: []cli.Arg{
		{Name: "vcs-server"},
		{Name: "repository"},
	},
}

func adminHookRepoDeleteRun(v cli.Values) error {
	path := fmt.Sprintf("/admin/repository/%s/%s", url.PathEscape(v.GetString("vcs-server")), url.PathEscape(v.GetString("repository")))
	if err := client.ServiceCallDELETE("hooks", path); err != nil {
		return err
	}
	return nil
}

var adminHooksRepoListCmd = cli.Command{
	Name:  "list",
	Short: "List repositories",
	Flags: []cli.Flag{
		{
			Name:    "pattern",
			Usage:   "Filter on repository name",
			Default: "",
		},
	},
}

func adminHooksRepoListRun(v cli.Values) (cli.ListResult, error) {
	btes, err := client.ServiceCallGET("hooks", "/v2/repository?filter="+v.GetString("pattern"))
	if err != nil {
		return nil, err
	}
	var repos []string
	if err := sdk.JSONUnmarshal(btes, &repos); err != nil {
		return nil, err
	}
	type Result struct {
		Repo string `cli:"vcs_server - repository"`
	}
	results := make([]Result, 0, len(repos))
	for _, r := range repos {
		results = append(results, Result{Repo: r})
	}
	return cli.AsListResult(results), nil
}
