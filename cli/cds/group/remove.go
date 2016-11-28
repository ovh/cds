package group

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdGroupRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "cds group remove <groupName>",
		Long:    ``,
		Run:     removeGroup,
		Aliases: []string{"delete", "rm", "del"},
	}

	return cmd
}

func removeGroup(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]

	err := sdk.RemoveGroup(name)
	if err != nil {
		sdk.Exit("%s\n", err)
	}
	fmt.Printf("OK\n")
}
