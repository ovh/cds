package application

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func applicationDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "cds application remove <projectKey> <applicationName>",
		Long:    ``,
		Run:     deleteApplication,
		Aliases: []string{"delete", "rm", "del"},
	}
	return cmd
}

func deleteApplication(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	name := args[1]

	err := sdk.DeleteApplication(projectKey, name)
	if err != nil {
		sdk.Exit("Error: cannot delete application %s (%s)\n", name, err)
	}

	fmt.Printf("Application %s deleted.\n", name)
}
