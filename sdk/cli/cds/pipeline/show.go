package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

func pipelineShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Short:   "cds pipeline show <projectKey> <pipelineName>",
		Long:    ``,
		Aliases: []string{"describe"},
		Run:     showPipeline,
	}
	return cmd
}

func showPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	pipelineName := args[1]
	p, err := sdk.GetPipeline(projectKey, pipelineName)
	if err != nil {
		sdk.Exit("Error: cannot retrieve pipeline informations: %s\n", err)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}
