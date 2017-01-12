package admin

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli/cds/admin/maintenance"
	"github.com/ovh/cds/cli/cds/admin/warning"
)

var (
	rootCmd = &cobra.Command{
		Use:   "admin",
		Short: "CDS Admin Management",
	}
)

func init() {
	rootCmd.AddCommand(warning.Cmd())
	rootCmd.AddCommand(maintenance.Cmd())
}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
