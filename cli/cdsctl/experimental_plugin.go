package main

import (
	"os"

	"github.com/rockbears/yaml"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
)

var experimentalPluginCmd = cli.Command{
	Name:  "plugin",
	Short: "CDS Experimental plugin commands",
}

func experimentalPlugin() *cobra.Command {
	return cli.NewCommand(experimentalPluginCmd, nil, []*cobra.Command{
		cli.NewCommand(pluginImportCmd, pluginImportFunc, nil),
	})
}

var pluginImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a plugin",
	Args: []cli.Arg{
		{
			Name: "file",
		},
	},
}

func pluginImportFunc(v cli.Values) error {
	b, err := os.ReadFile(v.GetString("file"))
	if err != nil {
		return cli.WrapError(err, "unable to read file %s", v.GetString("file"))
	}

	var expGPRCPlugin exportentities.GRPCPlugin
	if err := yaml.Unmarshal(b, &expGPRCPlugin); err != nil {
		return cli.WrapError(err, "unable to load file")
	}

	m := expGPRCPlugin.GRPCPlugin()

	if err := client.PluginImport(m); err != nil {
		return cli.WrapError(err, "unable to update plugin")
	}

	return nil
}
