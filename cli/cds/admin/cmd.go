package admin

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli/cds/admin/export"
	"github.com/ovh/cds/cli/cds/admin/importer"
	"github.com/ovh/cds/cli/cds/admin/maintenance"
	"github.com/ovh/cds/cli/cds/admin/plugin"
	"github.com/ovh/cds/cli/cds/admin/user"
)

var (
	rootCmd = &cobra.Command{
		Use:   "admin",
		Short: "CDS Admin Management",
	}
)

func init() {
	rootCmd.AddCommand(export.Cmd())
	rootCmd.AddCommand(importer.Cmd())
	rootCmd.AddCommand(maintenance.Cmd())
	rootCmd.AddCommand(plugin.Cmd())
	rootCmd.AddCommand(user.Cmd())
}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
