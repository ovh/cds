package application

import (
	"github.com/spf13/cobra"
)

// Cmd for pipeline operation
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application",
		Short:   "Application management",
		Long:    ``,
		Aliases: []string{"app"},
	}

	cmd.AddCommand(applicationDeleteCmd())
	cmd.AddCommand(applicationRenameCmd())
	cmd.AddCommand(applicationListCmd())
	cmd.AddCommand(applicationShowCmd())
	cmd.AddCommand(applicationVariableCmd)
	cmd.AddCommand(applicationGroupCmd)
	cmd.AddCommand(applicationPipelineCmd)
	cmd.AddCommand(applicationRepositoriesManagerCmd)
	cmd.AddCommand(cmdMetadata())

	return cmd
}
