package group

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdGroupList)
	Cmd.AddCommand(cmdGroupCreate)
	Cmd.AddCommand(cmdGroupUpdate)
	Cmd.AddCommand(cmdGroupDelete)
	Cmd.AddCommand(cmdGroupAddUser)
	Cmd.AddCommand(cmdGroupDeleteUser)
	Cmd.AddCommand(cmdGroupAddAdminUser)
	Cmd.AddCommand(cmdGroupDeleteAdminUser)
}

// Cmd group
var Cmd = &cobra.Command{
	Use:     "group",
	Short:   "Group commands: tatcli group --help",
	Long:    `Group commands: tatcli group <command>`,
	Aliases: []string{"g", "groups"},
}
