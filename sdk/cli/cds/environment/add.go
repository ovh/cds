package environment

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func environmentAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds environment add <projectKey> <environmentName>",
		Long:  ``,
		Run:   addEnvironment,
	}

	return cmd
}

func addEnvironment(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	name := args[1]

	err := sdk.AddEnvironment(projectKey, name)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	fmt.Printf("Environment %s created.\n", name)
}
