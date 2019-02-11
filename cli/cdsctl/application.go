package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var applicationCmd = cli.Command{
	Name:  "application",
	Short: "Manage CDS application",
}

func application() *cobra.Command {
	return cli.NewCommand(applicationCmd, nil, []*cobra.Command{
		cli.NewListCommand(applicationListCmd, applicationListRun, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(applicationShowCmd, applicationShowRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(applicationCreateCmd, applicationCreateRun, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(applicationDeleteCmd, applicationDeleteRun, nil, withAllCommandModifiers()...),
		applicationKey(),
		applicationVariable(),
		cli.NewCommand(applicationExportCmd, applicationExportRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(applicationImportCmd, applicationImportRun, nil, withAllCommandModifiers()...),
	})
}

var applicationListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS applications",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func applicationListRun(v cli.Values) (cli.ListResult, error) {
	apps, err := client.ApplicationList(v.GetString(_ProjectKey))
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
	app, err := client.ApplicationGet(v.GetString(_ProjectKey), v.GetString(_ApplicationName))
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
	a := &sdk.Application{Name: v.GetString(_ApplicationName)}
	return client.ApplicationCreate(v.GetString(_ProjectKey), a)
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
	err := client.ApplicationDelete(v.GetString(_ProjectKey), v.GetString(_ApplicationName))
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrApplicationNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	return err
}

var applicationImportCmd = cli.Command{
	Name:  "import",
	Short: "Import an application with a local filepath or an URL",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "path"},
	},
	Flags: []cli.Flag{
		{
			Type:    cli.FlagBool,
			Name:    "force",
			Usage:   "Override application if exists",
			Default: "false",
		},
	},
}

func applicationImportRun(c cli.Values) error {
	path := c.GetString("path")
	contentFile, format, err := exportentities.OpenPath(path)
	if err != nil {
		return err
	}
	defer contentFile.Close() //nolint
	formatStr, _ := exportentities.GetFormatStr(format)

	msgs, err := client.ApplicationImport(c.GetString(_ProjectKey), contentFile, formatStr, c.GetBool("force"))
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
			Type:    cli.FlagString,
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func applicationExportRun(c cli.Values) error {
	btes, err := client.ApplicationExport(c.GetString(_ProjectKey), c.GetString(_ApplicationName), c.GetString("format"))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}
