package application

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func applicationRenameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename",
		Short: "cds application rename <projectKey> <applicationName> <newApplicationName>",
		Long:  ``,
		Run:   renameApplication,
	}

	return cmd
}

func renameApplication(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	appName := args[1]
	newAppName := args[2]
	err := sdk.RenameApplication(projectKey, appName, newAppName)
	if err != nil {
		sdk.Exit("Error: cannot rename application: %s\n", err)
	}

	fmt.Printf("Application %s renamed to %s.\n", appName, newAppName)
}
