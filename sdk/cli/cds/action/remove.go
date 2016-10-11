package action

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdActionRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "cds action remove <actionName>",
		Long:    ``,
		Run:     removeAction,
		Aliases: []string{"delete", "rm", "del"},
	}
	return cmd
}

func removeAction(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]

	err := sdk.DeleteAction(name)
	if err != nil {
		sdk.Exit("%s\n", err)
	}
	fmt.Printf("OK\n")
}
