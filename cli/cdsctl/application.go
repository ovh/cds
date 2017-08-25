package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	applicationCmd = cli.Command{
		Name:  "application",
		Short: "Manage CDS application",
	}

	application = cli.NewCommand(applicationCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(applicationListCmd, applicationListRun, nil),
			cli.NewGetCommand(applicationShowCmd, applicationShowRun, nil),
			applicationKey,
		})
)

var applicationListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS applications",
	Args: []cli.Arg{
		{Name: "key"},
	},
}

func applicationListRun(v cli.Values) (cli.ListResult, error) {
	apps, err := client.ApplicationList(v["key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(apps), nil
}

var applicationShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS application",
	Args: []cli.Arg{
		{Name: "key"},
		{Name: "appName"},
	},
}

func applicationShowRun(v cli.Values) (interface{}, error) {
	app, err := client.ApplicationGet(v["key"], v["appName"])
	if err != nil {
		return nil, err
	}
	return *app, nil
}
