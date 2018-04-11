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
)

func admin() *cobra.Command {
	if cli.ShellMode {
		return cli.NewCommand(adminCmd, nil,
			[]*cobra.Command{
				adminServices,
				adminHooks,
				adminRepositories,
				adminVCS,
				usr,
				group,
				worker,
				health,
			})
	}
	return cli.NewCommand(adminCmd, nil,
		[]*cobra.Command{
			adminServices,
			adminHooks,
			adminRepositories,
			adminVCS,
		})
}
