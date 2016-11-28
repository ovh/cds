package environment

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func environmentUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds environment update <projectKey> <oldEnvironmentName> <newEnvironmentName>",
		Long:  ``,
		Run:   updateEnvironment,
	}

	return cmd
}

func updateEnvironment(cmd *cobra.Command, args []string) {

	if len(args) != 3 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	oldName := args[1]
	newName := args[2]

	err := sdk.UpdateEnvironment(projectKey, oldName, newName)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	fmt.Printf("Environment %s updated.\n", newName)
}
