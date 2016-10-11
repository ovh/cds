package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func pipelineDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "cds pipeline remove <projectKey> <pipelineName>",
		Long:    ``,
		Run:     deletePipeline,
		Aliases: []string{"delete", "rm", "del"},
	}
	return cmd
}

func deletePipeline(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	name := args[1]

	err := sdk.DeletePipeline(projectKey, name)
	if err != nil {
		sdk.Exit("Error: cannot delete pipeline %s (%s)\n", name, err)
	}

	fmt.Printf("Pipeline %s deleted.\n", name)
}
