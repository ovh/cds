package main

import (
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	adminPluginsCmd = cli.Command{
		Name:  "plugins",
		Short: "Manage CDS Plugins",
	}

	adminPlugins = cli.NewCommand(adminPluginsCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminPluginsListCmd, adminPluginsListFunc, nil),
			cli.NewCommand(adminPluginsImportCmd, adminPluginsImportFunc, nil),
			cli.NewCommand(adminPluginsExportCmd, adminPluginsExportFunc, nil),
			cli.NewDeleteCommand(adminPluginsDeleteCmd, adminPluginsDeleteFunc, nil),
		},
	)
)

var adminPluginsListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS Plugins",
}

func adminPluginsListFunc(v cli.Values) (cli.ListResult, error) {
	list, err := client.PluginsList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(list), nil
}

var adminPluginsImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a CDS Plugin",
	Args: []cli.Arg{
		{
			Name: "file",
		},
	},
}

func adminPluginsImportFunc(v cli.Values) error {
	b, err := ioutil.ReadFile(v.GetString("file"))
	if err != nil {
		return fmt.Errorf("unable to read file %s: %v", v.GetString("file"), err)
	}

	m := new(sdk.GRPCPlugin)
	if err := yaml.Unmarshal(b, m); err != nil {
		return fmt.Errorf("unable to load file: %v", err)
	}

	existing, err := client.PluginsGet(m.Name)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return fmt.Errorf("unable to get plugin: %v", err)
	}

	if existing == nil {
		if err := client.PluginAdd(m); err != nil {
			return fmt.Errorf("unable to add plugin: %v", err)
		}
		return nil
	}

	if err := client.PluginUpdate(m); err != nil {
		return fmt.Errorf("unable to update plugin: %v", err)
	}

	return nil
}

var adminPluginsExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a CDS Plugin",
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
}

func adminPluginsExportFunc(v cli.Values) error {
	p, err := client.PluginsGet(v.GetString("name"))
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("unable to marshal: %v", err)
	}

	fmt.Println(string(b))
	return nil
}

var adminPluginsDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS Plugin",
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
}

func adminPluginsDeleteFunc(v cli.Values) error {
	if err := client.PluginDelete(v.GetString("name")); err != nil {
		return fmt.Errorf("unable to delete plugin: %v", err)
	}
	return nil
}
