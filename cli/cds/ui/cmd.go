package ui

import (
	"fmt"

	"github.com/gizak/termui"
	"github.com/spf13/cobra"
)

// Cmd dashboard
var Cmd = &cobra.Command{
	Use:   "ui",
	Short: "cds ui",
	Run: func(cmd *cobra.Command, args []string) {
		runUI()
	},
}

func runUI() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("cds UI crashed :(\n%s\n", r)
			termui.Close()
		}
	}()

	ui := &Termui{}
	ui.init()
	ui.draw(0)

	defer termui.Close()
	termui.Loop()
}
