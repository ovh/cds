package action

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdActionAdd())
	Cmd.AddCommand(cmdActionRemove())
	Cmd.AddCommand(cmdActionList)
	Cmd.AddCommand(cmdActionShow())
	Cmd.AddCommand(cmdActionDoc())
}

// Cmd action
var Cmd = &cobra.Command{
	Use:     "action",
	Short:   "Action management",
	Long:    ``,
	Aliases: []string{"a"},
}
