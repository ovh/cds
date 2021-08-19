package main

import (
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
		adminCdnUnit(),
		cli.NewListCommand(adminCdnStatusCmd, adminCdnStatusRun, nil),
		cli.NewCommand(adminCdnMigFromCDSCmd, adminCdnMigFromCDS, nil),
		cli.NewCommand(adminCdnSyncBufferCmd, adminCdnSyncBuffer, nil),
	})
}

var adminCdnSyncBufferCmd = cli.Command{
	Name:    "sync-buffer",
	Short:   "run synchronization of cdn buffer",
	Example: "cdsctl admin cdn sync-buffer",
}

func adminCdnSyncBuffer(_ cli.Values) error {
	if _, err := client.ServiceCallPOST(sdk.TypeCDN, "/sync/buffer", nil); err != nil {
		return err
	}
	return nil
}

var adminCdnMigFromCDSCmd = cli.Command{
	Name:    "migrate",
	Short:   "run migration from cds to cdn",
	Example: "cdsctl admin cdn migrate",
}

func adminCdnStatusRun(_ cli.Values) (cli.ListResult, error) {
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

var adminCdnStatusCmd = cli.Command{
	Name:    "status",
	Short:   "display the status of cdn",
	Example: "cdsctl admin cdn status",
}

func adminCdnMigFromCDS(_ cli.Values) error {
	if _, err := client.ServiceCallPOST(sdk.TypeCDN, "/sync/projects", nil); err != nil {
		return err
	}
	return nil
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

func adminCdnCacheLogClearRun(_ cli.Values) error {
	return client.ServiceCallDELETE(sdk.TypeCDN, "/cache")
}

var adminCdnCacheLogStatusCmd = cli.Command{
	Name:    "status",
	Short:   "display the status of the cache log",
	Example: "cdsctl admin cdn cache status",
}

func adminCdnCacheLogStatusRun(_ cli.Values) (cli.ListResult, error) {
	btes, err := client.ServiceCallGET(sdk.TypeCDN, "/cache/status")
	if err != nil {
		return nil, err
	}
	ts := []sdk.MonitoringStatusLine{}
	if err := sdk.JSONUnmarshal(btes, &ts); err != nil {
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
