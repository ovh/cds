package system

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdSystemCacheClean)
	Cmd.AddCommand(cmdSystemCacheInfo)
}

// Cmd command
var Cmd = &cobra.Command{
	Use:     "system",
	Short:   "System commands (admin only): tatcli system --help",
	Long:    `System commands (admin only): tatcli system [<command>]`,
	Aliases: []string{"sys"},
}
