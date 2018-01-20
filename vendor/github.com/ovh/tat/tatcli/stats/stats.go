package stats

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdStatsCount)
	Cmd.AddCommand(cmdStatsDistribution)
	Cmd.AddCommand(cmdStatsDBStats)
	Cmd.AddCommand(cmdStatsDBServerStatus)
	Cmd.AddCommand(cmdStatsDBReplSetGetConfig)
	Cmd.AddCommand(cmdStatsDBReplSetGetStatus)
	Cmd.AddCommand(cmdStatsDBCollections)
	Cmd.AddCommand(cmdStatsDBSlowestQueries)
	Cmd.AddCommand(cmdStatsInstance)
}

// Cmd command
var Cmd = &cobra.Command{
	Use:     "stats",
	Short:   "Stats commands (admin only): tatcli stats --help",
	Long:    `Stats commands (admin only): tatcli stats [<command>]`,
	Aliases: []string{"stat"},
}
