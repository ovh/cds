package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalCmd = cli.Command{
	Name:    "experimental",
	Aliases: []string{"exp"},
	Short:   "CDS Experimental commands",
	Hidden:  true,
}

func experimentalCommands() []*cobra.Command {
	return []*cobra.Command{
		experimentalOrganization(),
		experimentalRegion(),
		experimentalProject(),
		experimentalRbac(),
		experimentalWorkerModel(),
	}
}

func experimental() *cobra.Command {
	return cli.NewCommand(experimentalCmd, nil, experimentalCommands())
}
