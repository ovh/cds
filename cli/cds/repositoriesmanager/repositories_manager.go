package repositoriesmanager

import "github.com/spf13/cobra"

// Cmd for pipeline operation
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reposmanager",
		Short: "List, add or remove repositories Manager (admin only)",
		Long:  ``,
	}

	cmd.AddCommand(listReposManagerCmd())
	cmd.AddCommand(addReposManagerCmd())

	return cmd
}
