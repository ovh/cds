package project

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdProjectRename() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename",
		Short: "cds project rename <projectUniqueKey> \"<projectNewName>\"",
		Long:  ``,
		Run:   renameProject,
	}

	return cmd
}

func renameProject(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	key := args[0]
	newName := strings.Join(args[1:len(args)], " ")

	err := sdk.RenameProject(key, newName)
	if err != nil {
		sdk.Exit("Error: cannot rename project %s (%s)\n", key, err)
	}

	fmt.Printf("OK\n")
}
