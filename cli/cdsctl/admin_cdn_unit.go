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
		cli.NewCommand(adminCdnUnitItemDeleteCmd, adminCdnItemUnitDelete, nil),
		cli.NewCommand(adminCdnUnitDeleteCmd, adminCdnUnitDelete, nil),
		cli.NewListCommand(adminCdnUnitListCmd, adminCdnUnitList, nil),
	})
}

var adminCdnUnitListCmd = cli.Command{
	Name:    "list",
	Short:   "list storage unit",
	Example: "cdsctl admin cdn unit list",
}

func adminCdnUnitList(_ cli.Values) (cli.ListResult, error) {
	bts, err := client.ServiceCallGET(sdk.TypeCDN, "/unit")
	if err != nil {
		return nil, err
	}
	var result []sdk.CDNUnitHandlerRequest
	if err := sdk.JSONUnmarshal(bts, &result); err != nil {
		return nil, err
	}
	return cli.AsListResult(result), nil
}

var adminCdnUnitItemDeleteCmd = cli.Command{
	Name:    "delete-items",
	Short:   "mark item as delete for given unit",
	Example: "cdsctl admin cdn unit delete-items <unit_id>",
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

var adminCdnUnitDeleteCmd = cli.Command{
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
