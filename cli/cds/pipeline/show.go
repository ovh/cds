package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

var withApplications bool

func pipelineShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Short:   "cds pipeline show <projectKey> <pipelineName>",
		Long:    ``,
		Aliases: []string{"describe"},
		Run:     showPipeline,
	}

	cmd.Flags().BoolVarP(&withApplications, "withApplications", "", false, "Show linked applications")

	return cmd
}

func showPipeline(cmd *cobra.Command, args []string) {
	var p *sdk.Pipeline
	var errG error
	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	pipelineName := args[1]

	if withApplications {
		p, errG = sdk.GetPipeline(projectKey, pipelineName, sdk.GetPipelineOptions.WithApplications)
	} else {
		p, errG = sdk.GetPipeline(projectKey, pipelineName)
	}

	if errG != nil {
		sdk.Exit("Error: cannot retrieve pipeline informations: %s\n", errG)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}
