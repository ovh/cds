package main

import (
	"fmt"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminCdnUnitCmd = cli.Command{
	Name:  "unit",
	Short: "Manage CDS CDN unit",
}

func adminCdnUnit() *cobra.Command {
	return cli.NewCommand(adminCdnUnitCmd, nil, []*cobra.Command{
		cli.NewCommand(adminCdnUnitItemDeleteCdm, adminCdnItemUnitDelete, nil),
		cli.NewCommand(adminCdnUnitDeleteCdm, adminCdnUnitDelete, nil),
	})
}

var adminCdnUnitItemDeleteCdm = cli.Command{
	Name:    "delete-item",
	Short:   "mark item as delete for given unit",
	Example: "cdsctl admin cdn unit delete <unit_id>",
	Args: []cli.Arg{
		{
			Name: "unit_id",
		},
	},
}

func adminCdnItemUnitDelete(v cli.Values) error {
	url := fmt.Sprintf("/unit/%s/item", v.GetString("unit_id"))
	if err := client.ServiceCallDELETE(sdk.TypeCDN, url); err != nil {
		return err
	}
	return nil
}

var adminCdnUnitDeleteCdm = cli.Command{
	Name:    "delete",
	Short:   "delete the given unit",
	Example: "cdsctl admin cdn unit delete <unit_id>",
	Args: []cli.Arg{
		{
			Name: "unit_id",
		},
	},
}

func adminCdnUnitDelete(v cli.Values) error {
	url := fmt.Sprintf("/unit/%s", v.GetString("unit_id"))
	if err := client.ServiceCallDELETE(sdk.TypeCDN, url); err != nil {
		return err
	}
	return nil
}
