package environment

import (
	"github.com/spf13/cobra"
)

// Cmd for pipeline operation
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environment",
		Short:   "Environment management",
		Long:    ``,
		Aliases: []string{"e", "env"},
	}

	cmd.AddCommand(environmentAddCmd())
	cmd.AddCommand(environmentUpdateCmd())
	cmd.AddCommand(environmentDeleteCmd())
	cmd.AddCommand(environmentListCmd())
	cmd.AddCommand(environmentShowCmd())
	cmd.AddCommand(environmentCloneCmd())
	cmd.AddCommand(environmentVariableCmd)
	cmd.AddCommand(environmentGroupCmd)

	return cmd
}
