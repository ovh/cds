package action

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

var cmdActionList = &cobra.Command{
	Use:     "list",
	Short:   "",
	Long:    ``,
	Aliases: []string{"ls"},
	Run:     listAction,
}

func listAction(cmd *cobra.Command, args []string) {
	actions, err := sdk.ListActions()
	if err != nil {
		sdk.Exit("%s\n", err)
	}

	for i := range actions {
		fmt.Printf("- %s\n", actions[i].Name)
	}
}
