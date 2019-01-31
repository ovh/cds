package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var applicationGroupCmd = cli.Command{
	Name:  "group",
	Short: "Manage CDS group linked to an application",
}

func applicationGroup() *cobra.Command {
	return cli.NewCommand(applicationGroupCmd, nil, []*cobra.Command{
		cli.NewCommand(applicationGroupImportCmd, applicationGroupImportRun, nil, withAllCommandModifiers()...),
	})
}

var applicationGroupImportCmd = cli.Command{
	Name:  "import",
	Short: "Import group linked to a CDS application",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
	Args: []cli.Arg{
		{Name: "path"},
	},
	Flags: []cli.Flag{
		{
			Name:  "force",
			Usage: "Use force flag to replace groups in your application",
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

func applicationGroupImportRun(v cli.Values) error {
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

	if sdk.IsURL(v.GetString("path")) {
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

	if _, err := client.ApplicationGroupsImport(v.GetString(_ProjectKey), v.GetString(_ApplicationName), reader, format, v.GetBool("force")); err != nil {
		return err
	}
	fmt.Printf("Groups imported in application %s with success\n", v.GetString(_ApplicationName))

	return nil
}
