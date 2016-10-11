package action

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

func cmdActionShow() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "show",
		Short: "cds action show <actionName>",
		Long:  ``,
		Run:   showAction,
	}

	return cmd
}

func showAction(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		sdk.Exit("Wrong usage. cds action run <actionName>\n")
	}
	aName := args[0]

	a, err := sdk.GetAction(aName)
	if err != nil {
		sdk.Exit("Error: cannot retrieve action %s: %s\n", aName, err)
	}

	data, err := yaml.Marshal(a)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}
