package user

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdUserAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds user add <username> <email> <fullname>",
		Long:  ``,
		Run:   addUser,
	}

	return cmd
}

func addUser(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]
	email := args[1]
	fullname := strings.Join(args[2:len(args)], " ")

	err := sdk.AddUser(name, fullname, email, "cdscli")
	if err != nil {
		sdk.Exit("Error: cannot add user %s (%s)\n", name, err)
	}
	fmt.Printf("Please check your email to activate your account\n")
}
