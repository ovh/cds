package group

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdGroupRename() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename",
		Short: "cds group rename <oldName> <newName>",
		Long:  ``,
		Run:   renameGroup,
	}

	return cmd
}

func renameGroup(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	oldName := args[0]
	newName := args[1]

	err := sdk.RenameGroup(oldName, newName)
	if err != nil {
		sdk.Exit("Error: cannot rename group %s (%s)\n", oldName, err)
	}
	fmt.Printf("OK\n")
}
