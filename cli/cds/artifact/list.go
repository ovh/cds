package artifact

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

var env string

func cmdArtifactList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "cds artifact list <projectName> <applicationName> <pipelineName> <tag>",
		Long:    ``,
		Run:     listArtifacts,
		Aliases: []string{"ls"},
	}
	cmd.Flags().StringVarP(&env, "env", "", "", "environment name")
	return cmd
}

func listArtifacts(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	project := args[0]
	appName := args[1]
	pipeline := args[2]
	tag := args[3]

	if env == "" {
		env = sdk.DefaultEnv.Name
	}

	arts, err := sdk.ListArtifacts(project, appName, pipeline, tag, env)
	if err != nil {
		sdk.Exit("Error: Cannot list artifacts in %s-%s-%s/%s (%s)\n", project, appName, pipeline, tag, err)
	}

	for _, a := range arts {
		fmt.Printf("- %s\n", a.Name)
	}
}
