package group

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

var cmdGroupList = &cobra.Command{
	Use:     "list",
	Short:   "",
	Long:    ``,
	Aliases: []string{"ls"},
	Run:     listGroup,
}

func listGroup(cmd *cobra.Command, args []string) {
	groups, err := sdk.ListGroups()
	if err != nil {
		sdk.Exit("%s\n", err)
	}

	for i := range groups {
		fmt.Printf("- %s \n", groups[i].Name)
	}
}
