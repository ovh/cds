package workflow

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var (
	rootCmd = &cobra.Command{
		Use:   "workflow",
		Short: "cds workflow",
	}

	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List workflow on current project: cds workflow list <project key>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			ws, err := sdk.WorkflowList(args[0])
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}
			sdk.Output("yaml", ws, fmt.Printf)
		},
	}

	showCmd = &cobra.Command{
		Use:   "show",
		Short: "Show a workflow on current project: cds workflow show <project key> <workflow name>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			ws, err := sdk.WorkflowGet(args[0], args[1])
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}
			sdk.Output("yaml", ws, fmt.Printf)
		},
	}
)

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
