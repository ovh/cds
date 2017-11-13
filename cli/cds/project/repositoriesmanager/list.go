package repositoriesmanager

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func listReposManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "cds project reposmanager list <projectKey>",
		Long:  ``,
		Run:   listReposManager,
	}

	return cmd
}

func listReposManager(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	rms, err := sdk.GetProjectReposManager(projectKey)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	for _, rm := range rms {
		fmt.Println(rm.Name)
	}
}
