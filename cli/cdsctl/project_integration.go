package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/spf13/cobra"
)

var ProjectIntegrationCmd = cli.Command{
	Name:  "integration",
	Short: "Manage CDS integration integrations",
}

func ProjectIntegration() *cobra.Command {
	return cli.NewCommand(ProjectIntegrationCmd, nil, []*cobra.Command{
		cli.NewListCommand(ProjectIntegrationListCmd, ProjectIntegrationListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(ProjectIntegrationDeleteCmd, ProjectIntegrationDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(ProjectIntegrationImportCmd, ProjectIntegrationImportFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(ProjectIntegrationExportCmd, ProjectIntegrationExportFunc, nil, withAllCommandModifiers()...),
	})
}

var ProjectIntegrationListCmd = cli.Command{
	Name:  "list",
	Short: "List integrations available on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func ProjectIntegrationListFunc(v cli.Values) (cli.ListResult, error) {
	pfs, err := client.ProjectIntegrationList(v.GetString(_ProjectKey))
	return cli.AsListResult(pfs), err
}

var ProjectIntegrationDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a integration configuration on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func ProjectIntegrationDeleteFunc(v cli.Values) error {
	return client.ProjectIntegrationDelete(v.GetString(_ProjectKey), v.GetString("name"))
}

var ProjectIntegrationImportCmd = cli.Command{
	Name:    "import",
	Short:   "Import a integration configuration on a project from a yaml file",
	Example: "cdsctl integration import MY-PROJECT file.yml",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "filename"},
	},
	Flags: []cli.Flag{
		{Name: "force", Type: cli.FlagBool},
	},
}

func ProjectIntegrationImportFunc(v cli.Values) error {
	f, err := os.Open(v.GetString("filename"))
	if err != nil {
		return fmt.Errorf("unable to open file %s: %v", v.GetString("filename"), err)
	}
	defer f.Close()
	_, err = client.ProjectIntegrationImport(v.GetString(_ProjectKey), f, filepath.Ext(v.GetString("filename")), v.GetBool("force"))
	return err
}

var ProjectIntegrationExportCmd = cli.Command{
	Name:    "export",
	Short:   "Export a integration configuration from a project to stdout",
	Example: "cdsctl integration export MY-PROJECT MY-INTEGRATION-NAME > file.yaml",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func ProjectIntegrationExportFunc(v cli.Values) error {
	pf, err := client.ProjectIntegrationGet(v.GetString(_ProjectKey), v.GetString("name"), false)
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
