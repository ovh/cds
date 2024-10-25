package main

import (
	"os"
	"strconv"

	"github.com/rockbears/yaml"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
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
	Flags: []cli.Flag{{
		Name: "force",
		Type: cli.FlagBool,
	}},
}

func pluginImportFunc(v cli.Values) error {
	force := v.GetBool("force")
	b, err := os.ReadFile(v.GetString("file"))
	if err != nil {
		return cli.WrapError(err, "unable to read file %s", v.GetString("file"))
	}

	var expGPRCPlugin exportentities.GRPCPlugin
	if err := yaml.Unmarshal(b, &expGPRCPlugin); err != nil {
		return cli.WrapError(err, "unable to load file")
	}

	m := expGPRCPlugin.GRPCPlugin()

	if err := client.PluginImport(m, cdsclient.WithQueryParameter("force", strconv.FormatBool(force))); err != nil {
		return cli.WrapError(err, "unable to update plugin")
	}

	return nil
}
