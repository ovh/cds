package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var cmdVersion = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the version of the worker binary",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CDS %s version:%s os:%s architecture:%s git.hash:%s build.time:%s\n", sdk.BINARY, sdk.VERSION, sdk.GOOS, sdk.GOARCH, sdk.GITHASH, sdk.BUILDTIME)
	},
}
