package presence

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdPresenceAdd)
	Cmd.AddCommand(cmdPresenceDelete)
	Cmd.AddCommand(cmdPresenceList)
}

// Cmd presence
var Cmd = &cobra.Command{
	Use:     "presence",
	Short:   "Presence commands: tatcli presence --help",
	Long:    `Presence commands: tatcli presence [<command>]`,
	Aliases: []string{"p"},
}
