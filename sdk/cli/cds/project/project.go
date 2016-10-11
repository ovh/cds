package project

import (
	"github.com/spf13/cobra"
	"github.com/ovh/cds/sdk/cli/cds/project/group"
	"github.com/ovh/cds/sdk/cli/cds/project/repositoriesmanager"
)

func init() {
	Cmd.AddCommand(cmdProjectAdd())
	Cmd.AddCommand(cmdProjectRename())
	Cmd.AddCommand(cmdProjectInfo())

	Cmd.AddCommand(cmdProjectRemove())
	Cmd.AddCommand(cmdProjectList)
	Cmd.AddCommand(group.CmdGroup)
	Cmd.AddCommand(CmdVariable)
	Cmd.AddCommand(repositoriesmanager.Cmd)
}

// Cmd project
var Cmd = &cobra.Command{
	Use:   "project",
	Short: "Project management",
	Long:  ``,
}
