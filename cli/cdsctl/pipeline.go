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
	pipelineCmd = cli.Command{
		Name:  "pipeline",
		Short: "Manage CDS pipeline",
	}

	pipeline = cli.NewCommand(pipelineCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(pipelineListCmd, pipelineListRun, nil),
			cli.NewCommand(pipelineExportCmd, pipelineExportRun, nil),
			cli.NewCommand(pipelineImportCmd, pipelineImportRun, nil),
		})
)

var pipelineListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS pipelines",
	Args: []cli.Arg{
		{Name: "project-key"},
	},
}

func pipelineListRun(v cli.Values) (cli.ListResult, error) {
	pipelines, err := client.PipelineList(v["project-key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(pipelines), nil
}

var pipelineExportCmd = cli.Command{
	Name:  "export",
	Short: "Export CDS pipeline",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "pipeline-name"},
	},
	Flags: []cli.Flag{
		{
			Name:  "format",
			Usage: "yml or json",
			IsValid: func(s string) bool {
				if s != "json" && s != "yml" {
					return false
				}
				return true
			},
			Kind:    reflect.String,
			Default: "yml",
		},
		{
			Name:  "with-permission",
			Usage: "true or false",
			IsValid: func(s string) bool {
				if s != "true" && s != "false" {
					return false
				}
				return true
			},
			Kind: reflect.Bool,
		},
	},
}

func pipelineExportRun(v cli.Values) error {
	btes, err := client.PipelineExport(v["project-key"], v["pipeline-name"], v.GetBool("with-permission"), v["format"])
	if err != nil {
		return err
	}
	fmt.Printf(string(btes))
	return nil
}

var pipelineImportCmd = cli.Command{
	Name:  "import",
	Short: "Import CDS pipeline",
	Long:  "PATH: Path or URL of pipeline to import",
	Args: []cli.Arg{
		{Name: "project-key"},
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

func pipelineImportRun(v cli.Values) error {
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

	msgs, err := client.PipelineImport(v["project-key"], btes, format, v.GetBool("force"))
	if err != nil {
		return err
	}
	for _, m := range msgs {
		fmt.Println(m)
	}
	return nil
}
