package environment

import (
	"fmt"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func environmentShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Short:   "cds environment show <projectKey> <environmentName>",
		Long:    ``,
		Aliases: []string{"describe"},
		Run:     showEnvironment,
	}

	return cmd
}

func showEnvironment(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	envName := args[1]
	p, err := sdk.GetEnvironment(projectKey, envName)
	if err != nil {
		sdk.Exit("Error: cannot retrieve environment informations: %s\n", err)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}
