package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/grpcplugin"
)

var adminPluginsCmd = cli.Command{
	Name:    "plugins",
	Short:   "Manage CDS Plugins",
	Aliases: []string{"plugin"},
}

func adminPlugins() *cobra.Command {
	return cli.NewCommand(adminPluginsCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminPluginsListCmd, adminPluginsListFunc, nil),
		cli.NewCommand(adminPluginsImportCmd, adminPluginsImportFunc, nil),
		cli.NewCommand(adminPluginsExportCmd, adminPluginsExportFunc, nil),
		cli.NewDeleteCommand(adminPluginsDeleteCmd, adminPluginsDeleteFunc, nil),
		cli.NewCommand(adminPluginsAddBinaryCmd, adminPluginsAddBinaryFunc, nil),
		cli.NewCommand(adminPluginsDocCmd, adminPluginsDocFunc, nil),
	})
}

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

	var expGPRCPlugin exportentities.GRPCPlugin
	if err := yaml.Unmarshal(b, &expGPRCPlugin); err != nil {
		return fmt.Errorf("unable to load file: %v", err)
	}

	m := expGPRCPlugin.GRPCPlugin()
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

var adminPluginsAddBinaryCmd = cli.Command{
	Name:  "binary-add",
	Short: "Add a binary",
	Args: []cli.Arg{
		{
			Name: "name",
		},
		{
			Name: "descriptor",
		},
		{
			Name: "filename",
		},
	},
}

func adminPluginsAddBinaryFunc(v cli.Values) error {
	p, err := client.PluginsGet(v.GetString("name"))
	if err != nil {
		return fmt.Errorf("unable to get plugin %s: %v", v.GetString("name"), err)
	}

	f, err := os.Open(v.GetString("filename"))
	if err != nil {
		return fmt.Errorf("unable to open file %s: %v", v.GetString("filename"), err)
	}

	fi, err := os.Stat(f.Name())
	if err != nil {
		return fmt.Errorf("unable to open file %s: %v", v.GetString("filename"), err)
	}

	b, err := ioutil.ReadFile(v.GetString("descriptor"))
	if err != nil {
		return fmt.Errorf("unable to read file %s: %v", v.GetString("file"), err)
	}

	var desc sdk.GRPCPluginBinary
	if err := yaml.Unmarshal(b, &desc); err != nil {
		return fmt.Errorf("unable to load file: %v", err)
	}

	desc.Name = filepath.Base(f.Name())
	desc.Perm = uint32(fi.Mode().Perm())
	desc.FileContent, err = ioutil.ReadFile(f.Name())
	if err != nil {
		return fmt.Errorf("unable to open file %s : %v", v.GetString("filename"), err)
	}

	desc.Size = int64(len(desc.FileContent))
	desc.MD5sum, err = sdk.FileMd5sum(v.GetString("filename"))
	if err != nil {
		return fmt.Errorf("unable to compute md5sum for file %s: %v", v.GetString("filename"), err)
	}

	desc.SHA512sum, err = sdk.FileSHA512sum(v.GetString("filename"))
	if err != nil {
		return fmt.Errorf("unable to compute sha512sum for file %s: %v", v.GetString("filename"), err)
	}

	return client.PluginAddBinary(p, &desc)
}

var adminPluginsDocCmd = cli.Command{
	Name:  "doc",
	Short: "Generate documentation in markdown for a plugin",
	Args: []cli.Arg{
		{
			Name: "path",
		},
	},
}

func adminPluginsDocFunc(v cli.Values) error {
	btes, errRead := ioutil.ReadFile(v.GetString("path"))
	if errRead != nil {
		return fmt.Errorf("Error while reading file: %s", errRead)
	}

	var expGPRCPlugin exportentities.GRPCPlugin
	if err := yaml.Unmarshal(btes, &expGPRCPlugin); err != nil {
		return fmt.Errorf("unable to load file: %v", err)
	}

	plg := expGPRCPlugin.GRPCPlugin()
	fmt.Println(grpcplugin.InfoMarkdown(*plg))

	return nil
}
