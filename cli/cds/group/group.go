package group

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdGroupAdd())
	Cmd.AddCommand(cmdGroupAddUser())
	Cmd.AddCommand(cmdGroupRemoveUser())
	Cmd.AddCommand(cmdGroupInfo())
	Cmd.AddCommand(cmdGroupRemove())
	Cmd.AddCommand(cmdGroupRename())
	Cmd.AddCommand(cmdGroupList)
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
