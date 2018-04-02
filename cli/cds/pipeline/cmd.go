package pipeline

import (
	"github.com/spf13/cobra"
)

const (
	statusFail = "status: Fail"
)

// Cmd for pipeline operation
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pipeline",
		Short:   "Manage and run pipelines",
		Long:    ``,
		Aliases: []string{"p", "pip"},
	}

	cmd.AddCommand(pipelineAddCmd())
	cmd.AddCommand(pipelineDeleteCmd())
	cmd.AddCommand(pipelineGroupCmd)
	cmd.AddCommand(pipelineHistoryCmd())
	cmd.AddCommand(pipelineRunCmd())
	cmd.AddCommand(pipelineRestartCmd())
	cmd.AddCommand(pipelineShowBuildCmd())
	cmd.AddCommand(pipelineCommitsCmd())
	cmd.AddCommand(pipelineStageCmd)
	cmd.AddCommand(pipelineHookCmd)
	cmd.AddCommand(pipelineParameterCmd)
	cmd.AddCommand(pipelineJobCmd())
	cmd.AddCommand(pipelineJoinedCmd())
	cmd.AddCommand(pipelineBuildCmd())
	cmd.AddCommand(exportCmd())
	cmd.AddCommand(importCmd())

	return cmd
}
