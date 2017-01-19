package maintenance

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	rootCmd = &cobra.Command{
		Use:   "maintenance",
		Short: "CDS Admin Maintenance Management (admin only)",
	}

	enableCmd = &cobra.Command{
		Use:   "enable",
		Short: "cds admin maintenance enable",
		Run: func(cmd *cobra.Command, args []string) {
			if confirm || cli.AskForConfirmation("Do you really want to enable maintenance mode ?") {
				_, _, err := sdk.Request("POST", "/admin/maintenance", nil)
				if err != nil {
					sdk.Exit("Error: %s\n", err)
				}
				fmt.Println("OK")
			} else {
				fmt.Println("Aborted")
			}
		},
	}

	disableCmd = &cobra.Command{
		Use:   "disable",
		Short: "cds admin maintenance disable",
		Run: func(cmd *cobra.Command, args []string) {
			if confirm || cli.AskForConfirmation("Do you really want to disable maintenance mode ?") {
				_, _, err := sdk.Request("DELETE", "/admin/maintenance", nil)
				if err != nil {
					sdk.Exit("Error: %s\n", err)
				}
				fmt.Println("OK")
			} else {
				fmt.Println("Aborted")
			}
		},
	}

	checkCmd = &cobra.Command{
		Use:   "check",
		Short: "cds admin maintenance check",
		Run: func(cmd *cobra.Command, args []string) {
			data, _, err := sdk.Request("GET", "/admin/maintenance", nil)
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}
			var m bool
			if err := json.Unmarshal(data, &m); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			if m {
				fmt.Println("Maintenance mode is on")
			} else {
				fmt.Println("Maintenance mode is off")
			}
		},
	}

	confirm bool
)

func init() {
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
	enableCmd.Flags().BoolVarP(&confirm, "yes", "y", false, "Automatic yes to prompt")
	disableCmd.Flags().BoolVarP(&confirm, "yes", "y", false, "Automatic yes to prompt")
}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
