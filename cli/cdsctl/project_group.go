package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

var projectGroupCmd = cli.Command{
	Name:    "group",
	Aliases: []string{"groups"},
	Short:   "Manage CDS group linked to a project",
}

func projectGroup() *cobra.Command {
	return cli.NewCommand(projectGroupCmd, nil, []*cobra.Command{
		cli.NewCommand(projectGroupImportCmd, projectGroupImportRun, nil, withAllCommandModifiers()...),
	})
}

var projectGroupImportCmd = cli.Command{
	Name:  "import",
	Short: "Import group linked to a CDS project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "path"},
	},
	Flags: []cli.Flag{
		{
			Name:  "force",
			Usage: "Use force flag to replace groups in your project",
			IsValid: func(s string) bool {
				if s != "true" && s != "false" {
					return false
				}
				return true
			},
			Default: "false",
			Type:    cli.FlagBool,
		},
	},
}

func projectGroupImportRun(v cli.Values) error {
	path := v.GetString("path")

	var reader io.ReadCloser
	var err error
	if sdk.IsURL(path) {
		reader, err = exportentities.OpenURL(path)
	} else {
		reader, err = exportentities.OpenFile(path)
	}
	if err != nil {
		return err
	}
	defer reader.Close() // nolint

	format, err := exportentities.GetFormatFromPath(path)
	if err != nil {
		return err
	}

	mods := []cdsclient.RequestModifier{
		cdsclient.ContentType(format.ContentType()),
	}
	if v.GetBool("force") {
		mods = append(mods, cdsclient.Force())
	}

	if _, err := client.ProjectGroupsImport(v.GetString(_ProjectKey), reader, mods...); err != nil {
		return err
	}
	fmt.Printf("Groups imported in project %s with success\n", v.GetString(_ProjectKey))

	return nil
}
