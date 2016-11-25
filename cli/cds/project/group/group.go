package group

import "github.com/spf13/cobra"

// CmdGroup Command to manage group management on project
var CmdGroup = &cobra.Command{
	Use:     "group",
	Short:   "",
	Long:    ``,
	Aliases: []string{"g"},
}

func init() {
	CmdGroup.AddCommand(cmdProjectAddGroup())
	CmdGroup.AddCommand(cmdProjectUpdateGroup())
	CmdGroup.AddCommand(cmdProjectRemoveGroup())
}
