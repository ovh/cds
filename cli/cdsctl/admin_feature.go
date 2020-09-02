package main

import (
	"fmt"
	"io/ioutil"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var adminFeaturesCmd = cli.Command{
	Name:    "features",
	Aliases: []string{"feature"},
	Short:   "Manage CDS feature flipping rules",
}

func adminFeatures() *cobra.Command {
	return cli.NewCommand(adminFeaturesCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminFeaturesListCmd, adminFeaturesListRun, nil),
		cli.NewCommand(adminFeatureExportCmd, adminFeatureExportRun, nil),
		cli.NewCommand(adminFeatureImportCmd, adminFeatureImportRun, nil),
		cli.NewDeleteCommand(adminFeatureDeleteCmd, adminFeatureDeleteRun, nil),
	})
}

// list command
var adminFeaturesListCmd = cli.Command{
	Name:    "list",
	Short:   "List all the features",
	Aliases: []string{"ls"},
}

func adminFeaturesListRun(v cli.Values) (cli.ListResult, error) {
	features, err := client.Features()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(features), nil
}

// Export command
var adminFeatureExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a feature as a yaml file",
	Aliases: []string{
		"show",
	},
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
}

func adminFeatureExportRun(v cli.Values) error {
	name := v.GetString("name")
	f, err := client.FeatureGet(name)
	if err != nil {
		return err
	}
	btes, err := yaml.Marshal(f)
	if err != nil {
		return err
	}

	fmt.Println(string(btes))
	return nil
}

// Import command
var adminFeatureImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a feature as a yaml file",
	Aliases: []string{
		"add",
	},
	Args: []cli.Arg{
		{
			Name: "file",
		},
	},
}

func adminFeatureImportRun(v cli.Values) error {
	btes, err := ioutil.ReadFile(v.GetString("file"))
	if err != nil {
		return err
	}
	var f sdk.Feature
	if err := yaml.Unmarshal(btes, &f); err != nil {
		return err
	}

	oldf, _ := client.FeatureGet(f.Name)
	if oldf.ID == 0 {
		return client.FeatureCreate(f)
	}
	return client.FeatureUpdate(f)
}

// Delete command
var adminFeatureDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete",
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
	Aliases: []string{
		"rm",
		"del",
	},
}

func adminFeatureDeleteRun(v cli.Values) error {
	name := v.GetString("name")
	return client.FeatureDelete(name)
}
