package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var previewCmd = cli.Command{
	Name:  "preview",
	Short: "CDS feature preview",
	Long:  "Preview commands should not be used in production. These commands are subject to breaking changes.",
}

func preview() *cobra.Command {
	return cli.NewCommand(previewCmd, nil, []*cobra.Command{
		cli.NewCommand(workflowV3ValidateCmd, workflowV3ValidateRun, nil),
		cli.NewCommand(workflowV3ConvertCmd, workflowV3ConvertRun, nil),
	})
}
