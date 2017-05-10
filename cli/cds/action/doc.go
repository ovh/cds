package action

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdActionDoc() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "doc",
		Short: "Generate Action Documenration: cds action doc <path-to-hclFile>",
		Long:  ``,
		Run:   docAction,
	}

	return cmd
}

func docAction(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage. cds action doc <path-to-hclFile>\n")
	}

	btes, errRead := ioutil.ReadFile(args[0])
	if errRead != nil {
		sdk.Exit("Error while reading file: %s", errRead)
	}

	action, errFrom := sdk.NewActionFromScript(btes)
	if errFrom != nil {
		sdk.Exit("Error loading file: %s", errFrom)
	}

	fmt.Println(sdk.ActionInfoMarkdown(action, path.Base(args[0])))
}
