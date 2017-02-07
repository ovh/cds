package pipeline

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func pipelineBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "build",
		Short:   "cds pipeline build <projectkey> <applicationName> <pipelineName> [envName] [buildID]",
		Long:    ``,
		Aliases: []string{},
		Run:     buildPipeline,
	}

	return cmd
}

func buildPipeline(cmd *cobra.Command, args []string) {

	if len(args) < 3 || len(args) > 5 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	appName := args[1]
	pipelineName := args[2]
	var buildNumber int
	var env string
	var err error
	if len(args) >= 4 {
		buildNumber, err = strconv.Atoi(args[3])
		if err != nil {
			// sdk.Exit("Error: buildID is not a number\n")
			// then it's the environment
			env = args[3]
			if len(args) == 5 {
				buildNumber, err = strconv.Atoi(args[4])
				if err != nil {
					sdk.Exit("Error: buildID is not a number\n")
				}
			}
		}
	}

	t, err := sdk.GetTestResults(projectKey, appName, pipelineName, env, buildNumber)
	if err != nil {
		sdk.Exit("Error: Cannot get tests results (%s)\n", err)
	}

	fmt.Printf("Tests results:\n")
	for _, s := range t.TestSuites {
		fmt.Printf("%s: %d Total, %d Failures, %d Errors\n", s.Name, s.Total, s.Failures, s.Errors)
	}
}
