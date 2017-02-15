package application

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func applicationAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds application add <projectKey> <applicationName>",
		Long:  ``,
		Run:   addApplication,
	}

	return cmd
}

func addApplication(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	name := args[1]

	err := sdk.AddApplication(projectKey, name)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	fmt.Printf("Application %s created.\n", name)
}
