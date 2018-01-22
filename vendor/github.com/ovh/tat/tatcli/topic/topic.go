package topic

import (
	"github.com/spf13/cobra"
)

var recursive bool

func init() {
	Cmd.AddCommand(cmdTopicList)
	Cmd.AddCommand(cmdTopicCreate)
	Cmd.AddCommand(cmdTopicDelete)
	Cmd.AddCommand(cmdTopicTruncate)
	Cmd.AddCommand(cmdTopicAddRoUser)
	Cmd.AddCommand(cmdTopicComputeLabels)
	Cmd.AddCommand(cmdTopicTruncateLabels)
	Cmd.AddCommand(cmdTopicComputeTags)
	Cmd.AddCommand(cmdTopicTruncateTags)
	Cmd.AddCommand(cmdTopicAllComputeLabels)
	Cmd.AddCommand(cmdTopicAllComputeTags)
	Cmd.AddCommand(cmdTopicAllComputeReplies)
	Cmd.AddCommand(cmdTopicAllSetParam)
	Cmd.AddCommand(cmdTopicAddRwUser)
	Cmd.AddCommand(cmdTopicAddAdminUser)
	Cmd.AddCommand(cmdTopicDeleteRoUser)
	Cmd.AddCommand(cmdTopicDeleteRwUser)
	Cmd.AddCommand(cmdTopicDeleteAdminUser)
	Cmd.AddCommand(cmdTopicAddRoGroup)
	Cmd.AddCommand(cmdTopicAddRwGroup)
	Cmd.AddCommand(cmdTopicAddAdminGroup)
	Cmd.AddCommand(cmdTopicDeleteRoGroup)
	Cmd.AddCommand(cmdTopicDeleteRwGroup)
	Cmd.AddCommand(cmdTopicDeleteAdminGroup)
	Cmd.AddCommand(cmdTopicAddParameter)
	Cmd.AddCommand(cmdTopicDeleteParameter)
	Cmd.AddCommand(cmdTopicParameter)
}

// Cmd topic
var Cmd = &cobra.Command{
	Use:     "topic",
	Short:   "Topic commands: tatcli topic --help",
	Long:    "Topic commands: tatcli topic [command]",
	Aliases: []string{"t"},
}
