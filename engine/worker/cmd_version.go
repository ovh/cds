package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var cmdVersion = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the version of the worker binary",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CDS Worker version:%s os:%s architecture:%s\n", sdk.VERSION, runtime.GOOS, runtime.GOARCH)
	},
}
