package user

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdUserReset() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "cds user reset <username> <email>",
		Long:  ``,
		Run:   resetUser,
	}

	return cmd
}

func resetUser(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]
	email := args[1]

	err := sdk.ResetUser(name, email, "cds user verify %s %s")
	if err != nil {
		sdk.Exit("Error: cannot reset user %s (%s)\n", name, err)
	}
	fmt.Printf("Please check your email to reset your password\n")
}
