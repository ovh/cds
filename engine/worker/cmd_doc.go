package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/doc"
)

func cmdDoc(root *cobra.Command) *cobra.Command {
	c := &cobra.Command{
		Use:    "doc",
		Short:  "generate hugo doc for building http://ovh.github.com/cds",
		Hidden: true,
		Run:    docCmd(root),
	}
	return c
}

func docCmd(root *cobra.Command) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if err := doc.GenerateDocumentation(root); err != nil {
			sdk.Exit(err.Error())
		}
	}
}
