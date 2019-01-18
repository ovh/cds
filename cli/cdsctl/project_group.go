package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
)

var projectGroupCmd = cli.Command{
	Name:  "group",
	Short: "Manage CDS group linked to a project",
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
	var reader io.ReadCloser
	defer func() {
		if reader != nil {
			reader.Close()
		}
	}()
	var format = "yaml"

	if strings.HasSuffix(v.GetString("path"), ".json") {
		format = "json"
	}

	if exportentities.IsURL(v.GetString("path")) {
		var err error
		reader, _, err = exportentities.OpenURL(v.GetString("path"), format)
		if err != nil {
			return err
		}
	} else {
		var err error
		reader, _, err = exportentities.OpenFile(v.GetString("path"))
		if err != nil {
			return err
		}
	}

	if _, err := client.ProjectGroupsImport(v.GetString(_ProjectKey), reader, format, v.GetBool("force")); err != nil {
		return err
	}
	fmt.Printf("Groups imported in project %s with success\n", v.GetString(_ProjectKey))

	return nil
}
