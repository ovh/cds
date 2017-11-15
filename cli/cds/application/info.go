package application

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func applicationShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info",
		Short:   "cds application info <projectKey> <applicationName>",
		Long:    ``,
		Aliases: []string{"describe", "show"},
		Run:     showApplication,
	}

	return cmd
}

func showApplication(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	appName := args[1]
	p, err := sdk.GetApplication(projectKey, appName, sdk.GetApplicationOptions.WithTriggers)
	if err != nil {
		sdk.Exit("Error: cannot retrieve application informations: %s\n", err)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}
