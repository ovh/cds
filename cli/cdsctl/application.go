package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

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
			cli.NewCommand(applicationExportCmd, applicationExportRun, nil),
			cli.NewCommand(applicationImportCmd, applicationImportRun, nil),
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
}

func applicationDeleteRun(v cli.Values) error {
	err := client.ApplicationDelete(v["project-key"], v["application-name"])
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrApplicationNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	return err
}

var applicationImportCmd = cli.Command{
	Name:  "import",
	Short: "Import an application",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "filename"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.Bool,
			Name:    "force",
			Usage:   "Override application if exists",
			Default: "false",
		},
	},
}

func applicationImportRun(c cli.Values) error {
	path := c.GetString("filename")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var format = "yaml"
	if strings.HasSuffix(path, ".json") {
		format = "json"
	}

	msgs, err := client.ApplicationImport(c.GetString("project-key"), f, format, c.GetBool("force"))
	if err != nil {
		return err
	}

	for _, s := range msgs {
		fmt.Println(s)
	}

	return nil
}

var applicationExportCmd = cli.Command{
	Name:  "export",
	Short: "Export an application",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "application-name"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.Bool,
			Name:    "with-permissions",
			Usage:   "Export permissions",
			Default: "false",
		},
		{
			Kind:    reflect.String,
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func applicationExportRun(c cli.Values) error {
	btes, err := client.ApplicationExport(c.GetString("project-key"), c.GetString("application-name"), c.GetBool("with-permissions"), c.GetString("format"))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}
