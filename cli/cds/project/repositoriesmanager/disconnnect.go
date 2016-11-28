package repositoriesmanager

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func disconnectReposManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "cds project reposmanager disconnect <project key> <repositories manager>",
		Long:  ``,
		Run:   disconnectReposManager,
	}

	return cmd
}

func disconnectReposManager(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	rmName := args[1]
	if err := sdk.DisconnectReposManager(projectKey, rmName); err != nil {
		sdk.Exit("✘ Error: %s\n", err)
	}
	fmt.Println("✔ Success")
}
