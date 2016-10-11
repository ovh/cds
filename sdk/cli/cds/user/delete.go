package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdUserDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "cds user remove <username>",
		Long:    ``,
		Run:     deleteUser,
		Aliases: []string{"delete", "rm", "del"},
	}

	return cmd
}

func deleteUser(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]

	err := sdk.DeleteUser(name)
	if err != nil {
		sdk.Exit("Error: cannot delete user %s (%s)\n", name, err)
	}
	fmt.Printf("User %s deleted\n", name)
}
