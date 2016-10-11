package user

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdUserAdd())
	Cmd.AddCommand(cmdUserList)
	Cmd.AddCommand(cmdUserReset())
	Cmd.AddCommand(cmdUserVerify())
	Cmd.AddCommand(cmdUserGenerate())
	Cmd.AddCommand(cmdUserUpdate())
	Cmd.AddCommand(cmdUserDelete())
}

// Cmd user
var Cmd = &cobra.Command{
	Use:     "user",
	Short:   "User management",
	Long:    ``,
	Aliases: []string{"u"},
}
