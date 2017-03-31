package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cmdVersion = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("CDS Worker version:", VERSION)
	},
}
