package environment

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func environmentDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "cds environment remove <projectKey> <environmentName>",
		Long:    ``,
		Run:     deleteEnvironment,
		Aliases: []string{"delete", "rm", "del"},
	}
	return cmd
}

func deleteEnvironment(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	name := args[1]

	err := sdk.DeleteEnvironment(projectKey, name)
	if err != nil {
		sdk.Exit("Error: cannot delete environment %s (%s)\n", name, err)
	}

	fmt.Printf("Environment %s deleted.\n", name)
}
