package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminCdnCmd = cli.Command{
	Name:  "cdn",
	Short: "Manage CDS CDN uService",
}

func adminCdn() *cobra.Command {
	return cli.NewCommand(adminCdnCmd, nil, []*cobra.Command{
		adminCdnCache(),
		adminCdnItem(),
		cli.NewListCommand(adminCdnStatusCmd, adminCdnStatusRun, nil),
	})
}

var adminCdnStatusCmd = cli.Command{
	Name:    "status",
	Short:   "display the status of cdn",
	Example: "cdsctl admin cdn status",
}

func adminCdnStatusRun(v cli.Values) (cli.ListResult, error) {
	services, err := client.ServicesByType(sdk.TypeCDN)
	if err != nil {
		return nil, err
	}
	status := sdk.MonitoringStatus{}
	for _, srv := range services {
		status.Lines = append(status.Lines, srv.MonitoringStatus.Lines...)
	}
	return cli.AsListResult(status.Lines), nil
}

var adminCdnCacheCmd = cli.Command{
	Name:  "cache",
	Short: "Manage CDS CDN Cache",
}

func adminCdnCache() *cobra.Command {
	return cli.NewCommand(adminCdnCacheCmd, nil, []*cobra.Command{
		cli.NewCommand(adminCdnCacheLogClearCmd, adminCdnCacheLogClearRun, nil),
		cli.NewListCommand(adminCdnCacheLogStatusCmd, adminCdnCacheLogStatusRun, nil),
	})
}

var adminCdnCacheLogClearCmd = cli.Command{
	Name:    "clear",
	Aliases: []string{"delete"},
	Short:   "clear the cache log",
	Example: "cdsctl admin cdn cache clear",
}

func adminCdnCacheLogClearRun(v cli.Values) error {
	return client.ServiceCallDELETE(sdk.TypeCDN, "/cache")
}

var adminCdnCacheLogStatusCmd = cli.Command{
	Name:    "status",
	Short:   "display the status of the cache log",
	Example: "cdsctl admin cdn cache status",
}

func adminCdnCacheLogStatusRun(v cli.Values) (cli.ListResult, error) {
	btes, err := client.ServiceCallGET(sdk.TypeCDN, "/cache/status")
	if err != nil {
		return nil, err
	}
	ts := []sdk.MonitoringStatusLine{}
	if err := json.Unmarshal(btes, &ts); err != nil {
		return nil, err
	}
	return cli.AsListResult(ts), nil
}

var adminCdnItemCmd = cli.Command{
	Name:    "item",
	Aliases: []string{"items"},
	Short:   "Manage CDS CDN Items",
}

func adminCdnItem() *cobra.Command {
	return cli.NewCommand(adminCdnItemCmd, nil, []*cobra.Command{
		cli.NewCommand(adminCdnItemSizeProjectCmd, adminCdnItemSizeProjectRun, nil),
	})
}

var adminCdnItemSizeProjectCmd = cli.Command{
	Name:    "projectsize",
	Short:   "Size used in octets by a project",
	Example: "cdsctl admin cdn item projectsize MYPROJ",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func adminCdnItemSizeProjectRun(v cli.Values) error {
	btes, err := client.ServiceCallGET(sdk.TypeCDN, "/size/item/project/"+v.GetString(_ProjectKey))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}
