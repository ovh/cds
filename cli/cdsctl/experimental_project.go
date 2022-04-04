package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var experimentalProjectCmd = cli.Command{
	Name:  "project",
	Short: "CDS Experimental project commands",
}

func experimentalProject() *cobra.Command {
	return cli.NewCommand(experimentalProjectCmd, nil, []*cobra.Command{
		projectVCS(),
	})
}
