package main

import (
	"fmt"
	"reflect"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
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
