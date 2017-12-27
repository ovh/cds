package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var cmdVersion = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CDS Worker version:%s os:%s architecture:%s\n", sdk.VERSION, sdk.OS, sdk.ARCH)
	},
}
