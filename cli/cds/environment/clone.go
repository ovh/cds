package environment

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

func environmentCloneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone",
		Short: "cds environment clone <projectKey> <environmentName> <newEnvironmentName>",
		Run:   cloneEnvironment,
	}

	return cmd
}

func cloneEnvironment(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	envName := args[1]
	newEnvName := args[2]
	p, err := sdk.CloneEnvironment(projectKey, envName, newEnvName)
	if err != nil {
		sdk.Exit("Error: cannot retrieve environment informations: %s\n", err)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}
