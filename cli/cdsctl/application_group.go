package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
)

var (
	applicationGroupCmd = cli.Command{
		Name:  "group",
		Short: "Manage CDS group linked to a pipeline",
	}

	applicationGroup = cli.NewCommand(applicationGroupCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(applicationGroupImportCmd, applicationGroupImportRun, nil),
		})
)

var applicationGroupImportCmd = cli.Command{
	Name:  "import",
	Short: "Import group linked to a CDS application",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "application-name"},
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
			Kind:    reflect.Bool,
		},
	},
}

func applicationGroupImportRun(v cli.Values) error {
	var btes []byte
	var format = "yaml"

	if strings.HasSuffix(v["path"], ".json") {
		format = "json"
	}

	isURL, _ := regexp.MatchString(`http[s]?:\/\/(.*)`, v["path"])
	if isURL {
		var err error
		btes, _, err = exportentities.ReadURL(v["path"], format)
		if err != nil {
			return err
		}
	} else {
		var err error
		btes, _, err = exportentities.ReadFile(v["path"])
		if err != nil {
			return err
		}
	}

	if _, err := client.ApplicationGroupsImport(v["project-key"], v["application-name"], btes, format, v.GetBool("force")); err != nil {
		return err
	}
	fmt.Printf("Groups imported in application %s with success\n", v["application-name"])

	return nil
}
