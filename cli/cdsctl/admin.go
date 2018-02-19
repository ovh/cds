package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	adminCmd = cli.Command{
		Name:  "admin",
		Short: "Manage CDS (admin only)",
	}

	admin = cli.NewCommand(adminCmd, nil,
		[]*cobra.Command{
			adminServices,
			adminHooks,
		})
)
