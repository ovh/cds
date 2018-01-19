package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/doc"
)

func cmdDoc(root *cobra.Command) *cobra.Command {
	c := &cobra.Command{
		Use:    "doc <generation-path>",
		Short:  "generate hugo doc for building http://ovh.github.com/cds",
		Hidden: true,
		Run:    docCmd(root),
	}
	return c
}

func docCmd(root *cobra.Command) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			cmd.Usage()
			os.Exit(1)
		}
		if err := doc.GenerateDocumentation(root, args[0], ""); err != nil {
			sdk.Exit(err.Error())
		}
	}
}
