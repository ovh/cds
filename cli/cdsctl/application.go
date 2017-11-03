package main

import (
	"fmt"
	"os"

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
			cli.NewDeleteCommand(applicationDeleteCmd, applicationDeleteRun, nil),
			applicationKey,
			applicationGroup,
			applicationVariable,
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
	Aliases: []string{"add"},
}

func applicationCreateRun(v cli.Values) error {
	a := &sdk.Application{Name: v["application-name"]}
	return client.ApplicationCreate(v["project-key"], a)
}

var applicationDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS application",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "application-name"},
	},
	Aliases: []string{"del", "rm", "remove"},
}

func applicationDeleteRun(v cli.Values) error {
	err := client.ApplicationDelete(v["project-key"], v["application-name"])
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrApplicationNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	return err
}
