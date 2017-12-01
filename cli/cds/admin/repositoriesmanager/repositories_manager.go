package repositoriesmanager

import "github.com/spf13/cobra"

// Cmd for pipeline operation
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reposmanager",
		Short: "CDS Admin Repositories Manager (admin only)",
	}

	cmd.AddCommand(listReposManagerCmd())

	return cmd
}
