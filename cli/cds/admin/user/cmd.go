package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

//Cmd returns the root cobra command for Users management
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Short:   "CDS Admin Users Management (admin only)",
		Aliases: []string{},
	}

	cmd.AddCommand(resetCmd)

	return cmd
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "cds admin users reset <username|all> <cds ui base url>",
	Run: func(cmd *cobra.Command, args []string) {
		if ok, err := sdk.IsAdmin(); !ok {
			if err != nil {
				fmt.Printf("Error : %v\n", err)
			}
			sdk.Exit("You are not allowed to run this command")
		}

		if len(args) != 2 {
			sdk.Exit("Wrong usage: %s\n", cmd.Short)
		}

		if args[0] == "all" {
			if cli.AskForConfirmation("Do you really want to reset all user password ?\n - Users will have to go to " + args[1]) {
				users, err := sdk.ListUsers()
				if err != nil {
					sdk.Exit("Error : %v\n", err)
				}
				for _, u := range users {
					if err := sdk.ResetUser(u.Username, u.Email, args[1]+"#/verify/%s/%s"); err != nil {
						fmt.Printf(" - %s: %s", u.Username, err.Error())
					}
				}
			} else {
				fmt.Println("Aborted")
			}
		} else {
			if cli.AskForConfirmation("Do you really want to reset user password ?\n - User will have to go to " + args[1]) {
				u, err := sdk.GetUser(args[0])
				if err != nil {
					sdk.Exit("Error : %v\n", err)
				}
				if err := sdk.ResetUser(u.Username, u.Email, args[1]+"#/verify/%s/%s"); err != nil {
					fmt.Printf(" - %s: %s", u.Username, err.Error())
				}
			} else {
				fmt.Println("Aborted")
			}
		}

	},
}
