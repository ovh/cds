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

	adminCommands = []*cobra.Command{
		adminServices,
		adminHooks,
		adminPlatformModels,
		adminMaintenance,
		adminPlugins,
		adminBroadcasts,
		adminErrors,
	}

	admin = cli.NewCommand(adminCmd, nil, adminCommands)

	adminShell = cli.NewCommand(adminCmd, nil, append(adminCommands,
		usr,
		group,
		worker,
		health,
	))
)
