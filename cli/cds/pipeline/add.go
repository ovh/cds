package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var pipelineType string

func pipelineAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds pipeline add <projectKey> <pipelineName>",
		Long:  ``,
		Run:   addPipeline,
	}

	cmd.Flags().StringVarP(&pipelineType, "type", "", "build", "Pipeline type {build,deployment,testing}")
	return cmd
}

func addPipeline(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	name := args[1]
	var t string

	found := false
	for i := range sdk.AvailablePipelineType {
		if sdk.AvailablePipelineType[i] == pipelineType {
			found = true
			t = sdk.AvailablePipelineType[i]
			break
		}
	}
	if !found {
		sdk.Exit("unknown pipeline type '%s'\n", pipelineType)
	}

	err := sdk.AddPipeline(name, projectKey, t, nil)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	fmt.Printf("Pipeline %s created.\n", name)
}
