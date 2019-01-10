package main

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
)

var environmentGroupCmd = cli.Command{
	Name:  "group",
	Short: "Manage CDS group linked to an environment",
}

func environmentGroup() *cobra.Command {
	return cli.NewCommand(environmentGroupCmd, nil, []*cobra.Command{
		cli.NewCommand(environmentGroupImportCmd, environmentGroupImportRun, nil, withAllCommandModifiers()...),
	})
}

var environmentGroupImportCmd = cli.Command{
	Name:  "import",
	Short: "Import group linked to a CDS environment",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "environment-name"},
		{Name: "path"},
	},
	Flags: []cli.Flag{
		{
			Name:  "force",
			Usage: "Use force flag to replace groups in your environment",
			IsValid: func(s string) bool {
				if s != "true" && s != "false" {
					return false
				}
				return true
			},
			Default: "false",
			Kind:    reflect.Bool,
		},
	},
}

func environmentGroupImportRun(v cli.Values) error {
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

	if _, err := client.EnvironmentGroupsImport(v[_ProjectKey], v["environment-name"], reader, format, v.GetBool("force")); err != nil {
		return err
	}
	fmt.Printf("Groups imported in environment %s with success\n", v["environment-name"])

	return nil
}
