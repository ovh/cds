package plugin

import (
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

//Cmd returns the root cobra command for plugin management
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plugin",
		Short:   "CDS Admin Plugin Management (admin only)",
		Aliases: []string{},
	}

	cmd.AddCommand(addPluginCmd)
	cmd.AddCommand(updatePluginCmd)
	cmd.AddCommand(deletePluginCmd)
	cmd.AddCommand(downloadPluginCmd)
	return cmd
}

var addPluginCmd = &cobra.Command{
	Use:   "add",
	Short: "cds plugin add <file>",
	Run: func(cmd *cobra.Command, args []string) {
		if ok, err := sdk.IsAdmin(); !ok {
			if err != nil {
				fmt.Printf("Error : %v\n", err)
			}
			sdk.Exit("You are not allowed to run this command")
		}

		if len(args) != 1 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}
		var err error
		for i := 0; i < 5; i++ {
			_, err = sdk.UploadPlugin(args[0], false)
			if err == nil {
				break
			}
		}
		if err != nil {
			sdk.Exit("Error: cannot add plugin %s (%s)\n", args[0], err)
		}
		fmt.Printf("OK\n")
	},
}

var updatePluginCmd = &cobra.Command{
	Use:   "update",
	Short: "cds plugin update <file>",
	Run: func(cmd *cobra.Command, args []string) {
		if ok, err := sdk.IsAdmin(); !ok {
			if err != nil {
				fmt.Printf("Error : %v\n", err)
			}
			sdk.Exit("You are not allowed to run this command")
		}

		if len(args) != 1 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}
		var err error
		for i := 0; i < 5; i++ {
			_, err = sdk.UploadPlugin(args[0], true)
			if err == nil {
				break
			}
		}
		if err != nil {
			sdk.Exit("Error: cannot add plugin %s (%s)\n", args[0], err)
		}
		fmt.Printf("OK\n")
	},
}

var deletePluginCmd = &cobra.Command{
	Use:   "delete",
	Short: "cds plugin delete <name>",
	Run: func(cmd *cobra.Command, args []string) {
		if ok, err := sdk.IsAdmin(); !ok {
			if err != nil {
				fmt.Printf("Error : %v\n", err)
			}
			sdk.Exit("You are not allowed to run this command")
		}

		if len(args) != 1 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}
		if err := sdk.DeletePlugin(args[0]); err != nil {
			sdk.Exit("Error: cannot delete plugin %s (%s)\n", args[0], err)
		}
		fmt.Printf("OK\n")
	},
}

var downloadPluginCmd = &cobra.Command{
	Use:   "download",
	Short: "cds plugin download <name>",
	Run: func(cmd *cobra.Command, args []string) {
		if ok, err := sdk.IsAdmin(); !ok {
			if err != nil {
				fmt.Printf("Error : %v\n", err)
			}
			sdk.Exit("You are not allowed to run this command")
		}

		if len(args) != 1 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}
		if err := sdk.DownloadPlugin(args[0], "."); err != nil {
			sdk.Exit("Error: cannot download plugin %s (%s)\n", args[0], err)
		}
		fmt.Printf("OK\n")
	},
}
