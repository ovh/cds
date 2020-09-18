package main

import (
	"encoding/json"

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
	})
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
	return client.ServiceCallDELETE("cdn", "/cache")
}

var adminCdnCacheLogStatusCmd = cli.Command{
	Name:    "status",
	Short:   "display the status of the cache log",
	Example: "cdsctl admin cdn cache status",
}

func adminCdnCacheLogStatusRun(v cli.Values) (cli.ListResult, error) {
	btes, err := client.ServiceCallGET("cdn", "/cache/status")
	if err != nil {
		return nil, err
	}
	ts := []sdk.MonitoringStatusLine{}
	if err := json.Unmarshal(btes, &ts); err != nil {
		return nil, err
	}
	return cli.AsListResult(ts), nil
}
