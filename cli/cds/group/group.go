package group

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdGroupAddUser())
	Cmd.AddCommand(cmdGroupRemoveUser())
	Cmd.AddCommand(cmdGroupInfo())
	Cmd.AddCommand(cmdGroupRename())
	Cmd.AddCommand(cmdGroupSetAdmin())
	Cmd.AddCommand(cmdGroupUnsetAdmin())
}

// Cmd group
var Cmd = &cobra.Command{
	Use:     "group",
	Short:   "Group management",
	Long:    ``,
	Aliases: []string{"g"},
}
