package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var adminCmd = cli.Command{
	Name:  "admin",
	Short: "Manage CDS (admin only)",
}

func adminCommands() []*cobra.Command {
	return []*cobra.Command{
		adminDatabase(),
		adminServices(),
		adminHooks(),
		adminPlatformModels(),
		adminMaintenance(),
		adminPlugins(),
		adminBroadcasts(),
		adminErrors(),
	}
}

func admin() *cobra.Command { return cli.NewCommand(adminCmd, nil, adminCommands()) }

func adminShell() *cobra.Command {
	return cli.NewCommand(adminCmd, nil, append(adminCommands(),
		usr(),
		group(),
		worker(),
		health(),
	))
}
