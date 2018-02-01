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
	environmentCmd = cli.Command{
		Name:  "environment",
		Short: "Manage CDS environment",
		Aliases: []string{
			"env",
		},
	}

	environment = cli.NewCommand(environmentCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(environmentListCmd, environmentListRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(environmentCreateCmd, environmentCreateRun, nil, withAllCommandModifiers()...),
			cli.NewDeleteCommand(environmentDeleteCmd, environmentDeleteRun, nil, withAllCommandModifiers()...),
			environmentKey,
			environmentVariable,
			environmentGroup,
			cli.NewCommand(environmentExportCmd, environmentExportRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(environmentImportCmd, environmentImportRun, nil, withAllCommandModifiers()...),
		})
)

var environmentListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS environments",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func environmentListRun(v cli.Values) (cli.ListResult, error) {
	apps, err := client.EnvironmentList(v[_ProjectKey])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(apps), nil
}

var environmentCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS environment",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "environment-name"},
	},
	Aliases: []string{"add"},
}

func environmentCreateRun(v cli.Values) error {
	env := &sdk.Environment{Name: v["environment-name"]}
	return client.EnvironmentCreate(v[_ProjectKey], env)
}

var environmentDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS environment",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "environment-name"},
	},
}

func environmentDeleteRun(v cli.Values) error {
	err := client.EnvironmentDelete(v[_ProjectKey], v["environment-name"])
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNoEnvironment) {
		fmt.Println(err.Error())
		os.Exit(0)
	}

	return err
}

var environmentImportCmd = cli.Command{
	Name:  "import",
	Short: "Import an environment",
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
			Usage:   "Override environment if exists",
			Default: "false",
		},
	},
}

func environmentImportRun(c cli.Values) error {
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

	msgs, err := client.EnvironmentImport(c.GetString(_ProjectKey), f, format, c.GetBool("force"))
	if err != nil {
		return err
	}

	for _, s := range msgs {
		fmt.Println(s)
	}

	return nil
}

var environmentExportCmd = cli.Command{
	Name:  "export",
	Short: "Export an environment",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "environment-name"},
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

func environmentExportRun(c cli.Values) error {
	btes, err := client.EnvironmentExport(c.GetString(_ProjectKey), c.GetString("environment-name"), c.GetBool("with-permissions"), c.GetString("format"))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}
