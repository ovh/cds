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
	projectGroupCmd = cli.Command{
		Name:  "group",
		Short: "Manage CDS group linked to a pipeline",
	}

	projectGroup = cli.NewCommand(projectGroupCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(projectGroupImportCmd, projectGroupImportRun, nil),
		})
)

var projectGroupImportCmd = cli.Command{
	Name:  "import",
	Short: "Import group linked to a CDS application",
	Args: []cli.Arg{
		{Name: "project-key"},
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

func projectGroupImportRun(v cli.Values) error {
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

	if _, err := client.ProjectGroupsImport(v["project-key"], btes, format, v.GetBool("force")); err != nil {
		return err
	}
	fmt.Printf("Groups imported in project %s with success\n", v["project-key"])

	return nil
}
