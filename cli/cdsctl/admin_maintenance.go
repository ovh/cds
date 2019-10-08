package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var adminMaintenancesCmd = cli.Command{
	Name:  "maintenance",
	Short: "Manage CDS maintenance",
}

func adminMaintenance() *cobra.Command {
	return cli.NewCommand(adminMaintenancesCmd, nil, []*cobra.Command{
		cli.NewCommand(adminMaintenanceEnableCmd, adminMaintenanceEnable, nil),
		cli.NewCommand(adminMaintenanceDisableCmd, adminMaintenanceDisable, nil),
	})
}

var adminMaintenanceEnableCmd = cli.Command{
	Name:  "enable",
	Short: "Enable CDS maintenance",
	Flags: []cli.Flag{
		{
			Name:    "hooks",
			Usage:   "provided to propagate to the hooks services",
			Default: "false",
			Type:    cli.FlagBool,
		},
	},
}

func adminMaintenanceEnable(v cli.Values) error {
	return client.Maintenance(true, v.GetBool("hooks"))
}

var adminMaintenanceDisableCmd = cli.Command{
	Name:  "disable",
	Short: "Disable CDS maintenance",
	Flags: []cli.Flag{
		{
			Name:    "hooks",
			Usage:   "provided to propagate to the hooks services",
			Default: "false",
			Type:    cli.FlagBool,
		},
	},
}

func adminMaintenanceDisable(v cli.Values) error {
	return client.Maintenance(false, v.GetBool("hooks"))
}
