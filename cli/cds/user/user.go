package user

import (
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func init() {
	Cmd.AddCommand(cmdUserInfo())
	Cmd.AddCommand(cmdUserList())
	Cmd.AddCommand(cmdUserReset())
	Cmd.AddCommand(cmdUserVerify())
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

func cmdUserInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "cds user info <username>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			name := args[0]

			u, err := sdk.GetUser(name)
			if err != nil {
				sdk.Exit("Error: %s", err.Error())
			}

			b, err := yaml.Marshal(u)
			if err != nil {
				sdk.Exit("Error: %s", err.Error())
			}

			fmt.Println(string(b))
		},
	}

	return cmd
}
