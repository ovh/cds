package template

import (
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
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
	cmd.AddCommand(deleteTemplateCmd)
	cmd.AddCommand(listTemplateCmd)
	cmd.AddCommand(updateTemplateCmd)

	return cmd
}

var addTemplateCmd = &cobra.Command{
	Use:   "add",
	Short: "cds templates add <file>",
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
	Short: "cds templates update <name> <file>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}
		var err error
		for i := 0; i < 5; i++ {
			_, err = sdk.UploadTemplate(args[1], true, args[0])
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
	Short: "cds templates delete <name>",
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

var listTemplateCmd = &cobra.Command{
	Use:   "list",
	Short: "cds templates list",
	Run: func(cmd *cobra.Command, args []string) {
		tmpls, err := sdk.ListTemplates()
		if err != nil {
			sdk.Exit("Error: cannot list templates: %s\n", err)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Type", "Description", "Actions"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")

		for _, t := range tmpls {
			table.Append([]string{t.Name, t.Type, t.Description, strings.Join(t.Actions, ",")})
		}

		table.Render()
	},
}
