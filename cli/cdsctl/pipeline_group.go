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
	pipelineGroupCmd = cli.Command{
		Name:  "group",
		Short: "Manage CDS group linked to a pipeline",
	}

	pipelineGroup = cli.NewCommand(pipelineGroupCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(pipelineGroupImportCmd, pipelineGroupImportRun, nil),
		})
)

var pipelineGroupImportCmd = cli.Command{
	Name:  "import",
	Short: "Import group linked to a CDS pipeline",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "pipeline-name"},
		{Name: "path"},
	},
	Flags: []cli.Flag{
		{
			Name:  "force",
			Usage: "Use force flag to update your pipeline",
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

func pipelineGroupImportRun(v cli.Values) error {
	var btes []byte
	var format = "yaml"

	if strings.HasSuffix(v["path"], ".json") {
		format = "json"
	} else if strings.HasSuffix(v["path"], ".hcl") {
		format = "hcl"
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

	if _, err := client.PipelineGroupsImport(v["project-key"], v["pipeline-name"], btes, format, v.GetBool("force")); err != nil {
		return err
	}
	fmt.Printf("Groups imported in pipeline %s with success\n", v["pipeline-name"])

	return nil
}
