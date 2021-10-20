package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

var projectIntegrationCmd = cli.Command{
	Name:    "integration",
	Aliases: []string{"integrations"},
	Short:   "Manage CDS integrations",
}

func projectIntegration() *cobra.Command {
	return cli.NewCommand(projectIntegrationCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectIntegrationListCmd, projectIntegrationListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(projectIntegrationDeleteCmd, projectIntegrationDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectIntegrationImportCmd, projectIntegrationImportFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectIntegrationExportCmd, projectIntegrationExportFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectIntegrationWorkerHooksExportCmd, projectIntegrationWorkerHooksExportFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectIntegrationWorkerHooksImportCmd, projectIntegrationWorkerHooksImportFunc, nil, withAllCommandModifiers()...),
	})
}

var projectIntegrationListCmd = cli.Command{
	Name:  "list",
	Short: "List integrations available on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectIntegrationListFunc(v cli.Values) (cli.ListResult, error) {
	pfs, err := client.ProjectIntegrationList(v.GetString(_ProjectKey))
	return cli.AsListResult(pfs), err
}

var projectIntegrationDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a integration configuration on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectIntegrationDeleteFunc(v cli.Values) error {
	return client.ProjectIntegrationDelete(v.GetString(_ProjectKey), v.GetString("name"))
}

var projectIntegrationImportCmd = cli.Command{
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

func projectIntegrationImportFunc(v cli.Values) error {
	f, err := os.Open(v.GetString("filename"))
	if err != nil {
		return cli.WrapError(err, "unable to open file %s", v.GetString("filename"))
	}
	defer f.Close()

	var mods []cdsclient.RequestModifier
	if v.GetBool("force") {
		mods = append(mods, cdsclient.Force())
	}

	_, err = client.ProjectIntegrationImport(v.GetString(_ProjectKey), f, mods...)
	return err
}

var projectIntegrationExportCmd = cli.Command{
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

func projectIntegrationExportFunc(v cli.Values) error {
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

var projectIntegrationWorkerHooksExportCmd = cli.Command{
	Name:  "worker-hooks-export",
	Short: "Export integration worker hooks available on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "integration"},
	},
}

func projectIntegrationWorkerHooksExportFunc(v cli.Values) error {
	res, err := client.ProjectIntegrationWorkerHooksList(v.GetString(_ProjectKey), v.GetString("integration"))
	if err != nil {
		return err
	}

	btes, err := exportentities.Marshal(res, exportentities.FormatYAML)
	if err != nil {
		return err
	}

	fmt.Println(string(btes))
	return err
}

var projectIntegrationWorkerHooksImportCmd = cli.Command{
	Name:  "worker-hooks-import",
	Short: "Import integration worker hooks on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "integration"},
		{Name: "filename"},
	},
}

func projectIntegrationWorkerHooksImportFunc(v cli.Values) error {
	f, err := os.Open(v.GetString("filename"))
	if err != nil {
		return cli.WrapError(err, "unable to open file %s", v.GetString("filename"))
	}
	defer f.Close()

	btes, err := ioutil.ReadAll(f)
	if err != nil {
		return cli.WrapError(err, "unable to read file %s", v.GetString("filename"))
	}

	var whs []sdk.WorkerHookProjectIntegrationModel
	if err := yaml.Unmarshal(btes, &whs); err != nil {
		return cli.WrapError(err, "unable to parse file %s", v.GetString("filename"))
	}

	err = client.ProjectIntegrationWorkerHooksImport(v.GetString(_ProjectKey), v.GetString("integration"), whs)
	if err != nil {
		return err
	}

	return nil
}
