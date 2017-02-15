package action

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var forceDelete bool

func cmdActionRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "cds action remove <actionName>",
		Long:    ``,
		Run:     removeAction,
		Aliases: []string{"delete", "rm", "del"},
	}

	cmd.Flags().BoolVarP(&forceDelete, "force", "", false, "delete action, exit 0 if action does not exist")

	return cmd
}

func removeAction(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]

	err := sdk.DeleteAction(name)
	if err != nil {
		if forceDelete && sdk.ErrorIs(err, sdk.ErrNoAction) {
			fmt.Printf("%s\n", err.Error())
			return
		}
		sdk.Exit("%s\n", err)
	}
	fmt.Printf("OK\n")
}
