package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
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
			cli.NewCommand(applicationCreateCmd, applicationCreateRun, nil),
			cli.NewCommand(applicationDeleteCmd, applicationDeleteRun, nil),
			applicationKey,
			applicationGroup,
		})
)

var applicationListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS applications",
	Args: []cli.Arg{
		{Name: "project-key"},
	},
}

func applicationListRun(v cli.Values) (cli.ListResult, error) {
	apps, err := client.ApplicationList(v["project-key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(apps), nil
}

var applicationShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS application",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "app-name"},
	},
}

func applicationShowRun(v cli.Values) (interface{}, error) {
	app, err := client.ApplicationGet(v["project-key"], v["app-name"])
	if err != nil {
		return nil, err
	}
	return *app, nil
}

var applicationCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS application",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "application-name"},
	},
}

func applicationCreateRun(v cli.Values) error {
	a := &sdk.Application{Name: v["application-name"]}
	return client.ApplicationCreate(v["project-key"], a)
}

var applicationDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS application",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "application-name"},
	},
}

func applicationDeleteRun(v cli.Values) error {
	return client.ApplicationDelete(v["project-key"], v["application-name"])
}
