package project

import (
	"github.com/ovh/cds/cli/cds/project/group"
	"github.com/ovh/cds/cli/cds/project/repositoriesmanager"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(cmdProjectRename())
	Cmd.AddCommand(cmdProjectInfo())
	Cmd.AddCommand(cmdMetadata())
	Cmd.AddCommand(group.CmdGroup)
	Cmd.AddCommand(repositoriesmanager.Cmd)
}

// Cmd project
var Cmd = &cobra.Command{
	Use:   "project",
	Short: "Project management",
	Long:  ``,
}
