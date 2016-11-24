package user

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdUserUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "update",
		Long: ``,
	}

	cmd.AddCommand(cmdUserUpdateFullName())
	cmd.AddCommand(cmdUserUpdateEmail())
	cmd.AddCommand(cmdUserUpdateUsername())
	return cmd
}

func cmdUserUpdateFullName() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fname",
		Short: "cds user update fname <username> <fullname>",
		Long:  ``,
		Run:   updateFname,
	}
	return cmd
}

func updateFname(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		sdk.Exit("Wrong usage: see %s\n", cmdUserUpdateFullName().Short)
	}
	userName := args[0]
	fullname := strings.Join(args[1:len(args)], " ")

	err := sdk.RenameUser(userName, fullname)
	if err != nil {
		sdk.Exit("Error: cannot rename user %s: %s\n", userName, err)
	}
	fmt.Println("OK")
}

func cmdUserUpdateEmail() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "email",
		Short: "cds user update email <username> <email>",
		Long:  ``,
		Run:   updateEmail,
	}
	return cmd
}

func updateEmail(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmdUserUpdateEmail().Short)
	}
	userName := args[0]
	email := args[1]

	err := sdk.UpdateUserEmail(userName, email)
	if err != nil {
		sdk.Exit("Error: cannot change user email address: %s\n", err)
	}
	fmt.Println("OK")
}

func cmdUserUpdateUsername() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "username",
		Short: "cds user update username <OldUsername> <NewUsername>",
		Long:  ``,
		Run:   updateUsername,
	}
	return cmd
}

func updateUsername(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmdUserUpdateUsername().Short)
	}
	oldUserName := args[0]
	newUserName := args[1]

	err := sdk.UpdateUsername(oldUserName, newUserName)
	if err != nil {
		sdk.Exit("Error: cannot change username: %s\n", err)
	}
	fmt.Println("OK")
}
