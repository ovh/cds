package main

import (
	"context"
	"os"

	"github.com/rockbears/yaml"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectRetentionCmd = cli.Command{
	Name:    "retention",
	Aliases: []string{""},
	Short:   "Manage CDS project workflow run retention",
}

func projectRetention() *cobra.Command {
	return cli.NewCommand(projectRetentionCmd, nil, []*cobra.Command{
		cli.NewCommand(projectRetentionImportCmd, projectRetentionImportFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectRetentionShowCmd, projectRetentionShowFunc, nil, withAllCommandModifiers()...),
	})
}

var projectRetentionImportCmd = cli.Command{
	Name:    "import",
	Aliases: []string{""},

	Short: "Update the project workflow run retention",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "filename"},
	},
}

func projectRetentionImportFunc(v cli.Values) error {
	btes, err := os.ReadFile(v.GetString("filename"))
	if err != nil {
		return cli.WrapError(err, "unable to open file %s", v.GetString("filename"))
	}

	var content sdk.Retentions
	if err := yaml.Unmarshal(btes, &content); err != nil {
		return cli.WrapError(err, "unable to parse file %s", v.GetString("filename"))
	}

	currentProjectRetention, err := client.ProjectRunRetentionGet(context.Background(), v.GetString(_ProjectKey))
	if err != nil {
		return err
	}
	currentProjectRetention.Retentions = content

	if err := client.ProjectRunRetentionImport(context.Background(), v.GetString(_ProjectKey), *currentProjectRetention); err != nil {
		return err
	}
	return nil
}

var projectRetentionShowCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},

	Short: "Retrive the current project run retention",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectRetentionShowFunc(v cli.Values) (interface{}, error) {
	pr, err := client.ProjectRunRetentionGet(context.Background(), v.GetString(_ProjectKey))
	return pr, err
}
