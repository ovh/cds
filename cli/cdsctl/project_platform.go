package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/spf13/cobra"
)

var (
	projectPlatformCmd = cli.Command{
		Name:  "platform",
		Short: "Manage CDS project platforms",
	}

	projectPlatform = cli.NewCommand(projectPlatformCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(projectPlatformListCmd, projectPlatformListFunc, nil, withAllCommandModifiers()...),
			cli.NewDeleteCommand(projectPlatformDeleteCmd, projectPlatformDeleteFunc, nil, withAllCommandModifiers()...),
			cli.NewCommand(projectPlatformImportCmd, projectPlatformImportFunc, nil, withAllCommandModifiers()...),
			cli.NewCommand(projectPlatformExportCmd, projectPlatformExportFunc, nil, withAllCommandModifiers()...),
		})
)

var projectPlatformListCmd = cli.Command{
	Name:  "list",
	Short: "List platforms available on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectPlatformListFunc(v cli.Values) (cli.ListResult, error) {
	pfs, err := client.ProjectPlatformList(v.GetString(_ProjectKey))
	return cli.AsListResult(pfs), err
}

var projectPlatformDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a platform configuration on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectPlatformDeleteFunc(v cli.Values) error {
	return client.ProjectPlatformDelete(v.GetString(_ProjectKey), v.GetString("name"))
}

var projectPlatformImportCmd = cli.Command{
	Name:    "import",
	Short:   "Import a platform configuration on a project from a yaml file",
	Example: "cdsctl project platform import MY-PROJECT file.yml",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "filename"},
	},
	Flags: []cli.Flag{
		{Name: "force", Kind: reflect.Bool},
	},
}

func projectPlatformImportFunc(v cli.Values) error {
	f, err := os.Open(v.GetString("filename"))
	if err != nil {
		return fmt.Errorf("unable to open file %s: %v", v.GetString("filename"), err)
	}
	defer f.Close()
	_, err = client.ProjectPlatformImport(v.GetString(_ProjectKey), f, filepath.Ext(v.GetString("filename")), v.GetBool("force"))
	return err
}

var projectPlatformExportCmd = cli.Command{
	Name:    "export",
	Short:   "Export a platform configuration from a project to stdout",
	Example: "cdsctl project platform export MY-PROJECT MY-PLATFORM-NAME > file.yaml",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectPlatformExportFunc(v cli.Values) error {
	pf, err := client.ProjectPlatformGet(v.GetString(_ProjectKey), v.GetString("name"), false)
	if err != nil {
		return err
	}

	btes, err := exportentities.Marshal(pf, exportentities.FormatYAML)
	if err != nil {
		return err
	}

	fmt.Println(string(btes))
	return nil
}
