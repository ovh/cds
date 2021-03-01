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
		cli.NewCommand(adminCdnUnitDeleteCdm, adminCdnUnitDelete, nil),
	})
}

var adminCdnUnitDeleteCdm = cli.Command{
	Name:    "delete",
	Short:   "mark a unit as delete",
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
