package template

import (
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

//Cmd returns the root cobra command for Template management
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "templates",
		Short:   "templates management (admin only)",
		Long:    ``,
		Aliases: []string{},
	}

	cmd.AddCommand(addTemplateCmd)
	cmd.AddCommand(updateTemplateCmd)
	cmd.AddCommand(deleteTemplateCmd)
	cmd.AddCommand(downloadTemplateCmd)
	return cmd
}

var addTemplateCmd = &cobra.Command{
	Use:   "add",
	Short: "cds Template add <file>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}
		var err error
		for i := 0; i < 5; i++ {
			_, err = sdk.UploadTemplate(args[0], false, "")
			if err == nil {
				break
			}
		}
		if err != nil {
			sdk.Exit("Error: cannot add Template %s (%s)\n", args[0], err)
		}
		fmt.Printf("OK\n")
	},
}

var updateTemplateCmd = &cobra.Command{
	Use:   "update",
	Short: "cds Template update <name> <file>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}
		var err error
		for i := 0; i < 5; i++ {
			_, err = sdk.UploadTemplate(args[0], true, args[1])
			if err == nil {
				break
			}
		}
		if err != nil {
			sdk.Exit("Error: cannot add Template %s (%s)\n", args[0], err)
		}
		fmt.Printf("OK\n")
	},
}

var deleteTemplateCmd = &cobra.Command{
	Use:   "delete",
	Short: "cds Template delete <name>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}
		if err := sdk.DeleteTemplate(args[0]); err != nil {
			sdk.Exit("Error: cannot delete Template %s (%s)\n", args[0], err)
		}
		fmt.Printf("OK\n")
	},
}

var downloadTemplateCmd = &cobra.Command{
	Use:   "download",
	Short: "cds Template download <name>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}
		if err := sdk.DownloadTemplate(args[0], "."); err != nil {
			sdk.Exit("Error: cannot download Template %s (%s)\n", args[0], err)
		}
		fmt.Printf("OK\n")
	},
}
