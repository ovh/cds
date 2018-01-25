package message

import (
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(cmdMessageAdd)
	Cmd.AddCommand(cmdMessageReply)
	Cmd.AddCommand(cmdMessageDelete)
	Cmd.AddCommand(cmdMessageDeleteBulk)
	Cmd.AddCommand(cmdMessageUpdate)
	Cmd.AddCommand(cmdMessageConcat)
	Cmd.AddCommand(cmdMessageMove)
	Cmd.AddCommand(cmdMessageTask)
	Cmd.AddCommand(cmdMessageUntask)
	Cmd.AddCommand(cmdMessageLike)
	Cmd.AddCommand(cmdMessageUnlike)
	Cmd.AddCommand(cmdMessageVoteUP)
	Cmd.AddCommand(cmdMessageVoteDown)
	Cmd.AddCommand(cmdMessageUnVoteUP)
	Cmd.AddCommand(cmdMessageUnVoteDown)
	Cmd.AddCommand(cmdMessageLabel)
	Cmd.AddCommand(cmdMessageUnlabel)
	Cmd.AddCommand(cmdMessageRelabel)
	Cmd.AddCommand(cmdMessageList)
}

// Cmd message
var Cmd = &cobra.Command{
	Use:     "message",
	Short:   "Manipulate messages: tatcli message --help",
	Long:    `Manipulate messages: tatcli message <command>`,
	Aliases: []string{"m", "msg"},
}
