package group

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdGroupAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds group add <groupName>",
		Long:  ``,
		Run:   addGroup,
	}

	return cmd
}

func addGroup(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]

	err := sdk.AddGroup(name)
	if err != nil {
		sdk.Exit("Error: cannot add group %s (Reason: %s)\n", name, err)
	}
	fmt.Printf("OK\n")
}
