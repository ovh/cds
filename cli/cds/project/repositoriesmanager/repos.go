package repositoriesmanager

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func getReposFromReposManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repos",
		Short: "cds project reposmanager repos <projectKey> <repositories manager>",
		Long:  ``,
		Run:   getRepos,
	}

	return cmd
}

func getRepos(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	rmName := args[1]

	repos, err := sdk.GetProjectReposFromReposManager(projectKey, rmName)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	for _, r := range repos {
		fmt.Printf("%s %s %s %s\n", r.Name, r.Slug, r.Fullname, r.URL)
	}
}
