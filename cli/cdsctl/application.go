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
			cli.NewListCommand(applicationListCmd, applicationListRun, nil, withAllCommandModifiers()...),
			cli.NewGetCommand(applicationShowCmd, applicationShowRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(applicationCreateCmd, applicationCreateRun, nil, withAllCommandModifiers()...),
			cli.NewDeleteCommand(applicationDeleteCmd, applicationDeleteRun, nil, withAllCommandModifiers()...),
			applicationKey,
			applicationGroup,
			applicationVariable,
			cli.NewCommand(applicationExportCmd, applicationExportRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(applicationImportCmd, applicationImportRun, nil, withAllCommandModifiers()...),
		})
)

var applicationListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS applications",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func applicationListRun(v cli.Values) (cli.ListResult, error) {
	apps, err := client.ApplicationList(v[_ProjectKey])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(apps), nil
}

var applicationShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS application",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
}

func applicationShowRun(v cli.Values) (interface{}, error) {
	app, err := client.ApplicationGet(v[_ProjectKey], v[_ApplicationName])
	if err != nil {
		return nil, err
	}
	return *app, nil
}

var applicationCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS application",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: _ApplicationName},
	},
	Aliases: []string{"add"},
}

func applicationCreateRun(v cli.Values) error {
	a := &sdk.Application{Name: v[_ApplicationName]}
	return client.ApplicationCreate(v[_ProjectKey], a)
}

var applicationDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS application",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
}

func applicationDeleteRun(v cli.Values) error {
	err := client.ApplicationDelete(v[_ProjectKey], v[_ApplicationName])
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrApplicationNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	return err
}

var applicationImportCmd = cli.Command{
	Name:  "import",
	Short: "Import an application",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
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

	msgs, err := client.ApplicationImport(c.GetString(_ProjectKey), f, format, c.GetBool("force"))
	if err != nil {
		if msgs != nil {
			for _, msg := range msgs {
				fmt.Println(msg)
			}
		}
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
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
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
	btes, err := client.ApplicationExport(c.GetString(_ProjectKey), c.GetString(_ApplicationName), c.GetBool("with-permissions"), c.GetString("format"))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}
