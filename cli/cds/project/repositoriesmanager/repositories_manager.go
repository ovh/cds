package repositoriesmanager

import "github.com/spf13/cobra"

//Cmd is the root command for all reposmanager actions
var Cmd = &cobra.Command{
	Use:     "reposmanager",
	Short:   "",
	Long:    ``,
	Aliases: []string{},
}

func init() {
	Cmd.AddCommand(listReposManagerCmd())
	Cmd.AddCommand(addApplicationCmd())
	Cmd.AddCommand(connectReposManagerCmd())
	Cmd.AddCommand(disconnectReposManagerCmd())
	Cmd.AddCommand(getReposFromReposManagerCmd())
	Cmd.AddCommand(getCommitsCmd())
}
