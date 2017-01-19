package warning

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	rootCmd = &cobra.Command{
		Use:   "warning",
		Short: "CDS Admin Warning Management (admin only)",
	}

	truncateCmd = &cobra.Command{
		Use:   "truncate",
		Short: "cds admin warning truncate",
		Run: func(cmd *cobra.Command, args []string) {
			if confirm || cli.AskForConfirmation("Do you really want to truncate all warnings ?") {
				_, _, err := sdk.Request("DELETE", "/admin/warning", nil)
				if err != nil {
					sdk.Exit("Error: %s\n", err)
				}
				fmt.Println("OK")
			} else {
				fmt.Println("Aborted")
			}
		},
	}

	confirm bool
)

func init() {
	rootCmd.AddCommand(truncateCmd)
	truncateCmd.Flags().BoolVarP(&confirm, "yes", "y", false, "Automatic yes to prompt")
}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
