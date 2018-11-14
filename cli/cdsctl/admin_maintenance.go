package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	adminMaintenancesCmd = cli.Command{
		Name:  "maintenance",
		Short: "Manage CDS maintenance",
	}

	adminMaintenance = cli.NewCommand(adminMaintenancesCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(adminMaintenanceEnableCmd, adminMaintenanceEnable, nil),
			cli.NewCommand(adminMaintenanceDisableCmd, adminMaintenanceDisable, nil),
		})
)

var adminMaintenanceEnableCmd = cli.Command{
	Name:  "enable",
	Short: "Enable CDS maintenance",
}

func adminMaintenanceEnable(v cli.Values) error {
	return client.Maintenance(true)
}

var adminMaintenanceDisableCmd = cli.Command{
	Name:  "disable",
	Short: "Disable CDS maintenance",
}

func adminMaintenanceDisable(v cli.Values) error {
	return client.Maintenance(false)
}
