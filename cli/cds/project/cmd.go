package project

import (
	"github.com/ovh/cds/cli/cds/project/group"
	"github.com/ovh/cds/cli/cds/project/repositoriesmanager"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(cmdProjectAdd())
	Cmd.AddCommand(cmdProjectRename())
	Cmd.AddCommand(cmdProjectInfo())
	Cmd.AddCommand(cmdMetadata())
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
