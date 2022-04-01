package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var projectExperimentalCmd = cli.Command{
	Name:    "experimental",
	Aliases: []string{"exp"},
	Short:   "CDS Experimental commands",
}

func projectExperimental() *cobra.Command {
	return cli.NewCommand(projectExperimentalCmd, nil, []*cobra.Command{
		projectVCS(),
	})
}
