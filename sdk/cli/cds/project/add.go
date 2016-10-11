package project

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdProjectAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds project add <projectUniqueKey> \"<projectName>\" <groupName>",
		Long:  ``,
		Run:   addProject,
	}

	return cmd
}

func addProject(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	key := args[0]
	name := args[1]
	groupName := args[2]

	err := sdk.AddProject(name, key, groupName)
	if err != nil {
		sdk.Exit("Error: cannot add project %s (%s)\n", name, err)
	}

	fmt.Printf("OK\n")
}
