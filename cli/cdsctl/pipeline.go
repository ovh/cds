package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var (
	pipelineCmd = cli.Command{
		Name:  "pipeline",
		Short: "Manage CDS pipeline",
	}

	pipeline = cli.NewCommand(pipelineCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(pipelineListCmd, pipelineListRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(pipelineCreateCmd, pipelineCreateRun, nil, withAllCommandModifiers()...),
			cli.NewDeleteCommand(pipelineDeleteCmd, pipelineDeleteRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(pipelineExportCmd, pipelineExportRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(pipelineImportCmd, pipelineImportRun, nil, withAllCommandModifiers()...),
			pipelineGroup,
		})
)

var pipelineListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS pipelines",
	Ctx: []cli.Arg{
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

var pipelineCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS pipeline",
	Ctx: []cli.Arg{
		{Name: "project-key"},
	},
	Args: []cli.Arg{
		{Name: "pipeline-name"},
	},
	Flags: []cli.Flag{
		{
			Name:  "type",
			Usage: `Pipeline type {build,deployment,testing} (default "build")`,
			IsValid: func(s string) bool {
				if s != "" && s != "build" && s != "deployment" && s != "testing" {
					return false
				}
				return true
			},
			Default: "build",
			Kind:    reflect.String,
		},
	},
	Aliases: []string{"add"},
}

func pipelineCreateRun(v cli.Values) error {
	pip := &sdk.Pipeline{Name: v["pipeline-name"], Type: v.GetString("type")}
	return client.PipelineCreate(v["project-key"], pip)
}

var pipelineExportCmd = cli.Command{
	Name:  "export",
	Short: "Export CDS pipeline",
	Ctx: []cli.Arg{
		{Name: "project-key"},
	},
	Args: []cli.Arg{
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
	Ctx: []cli.Arg{
		{Name: "project-key"},
	},
	Args: []cli.Arg{
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
	var reader io.ReadCloser
	defer func() {
		if reader != nil {
			reader.Close()
		}
	}()
	var format = "yaml"

	if strings.HasSuffix(v["path"], ".json") {
		format = "json"
	} else if strings.HasSuffix(v["path"], ".hcl") {
		format = "hcl"
	}

	isURL, _ := regexp.MatchString(`http[s]?:\/\/(.*)`, v["path"])
	if isURL {
		var err error
		reader, _, err = exportentities.OpenURL(v["path"], format)
		if err != nil {
			return err
		}
	} else {
		var err error
		reader, _, err = exportentities.OpenFile(v["path"])
		if err != nil {
			return err
		}
	}

	msgs, err := client.PipelineImport(v["project-key"], reader, format, v.GetBool("force"))
	if err != nil {
		return err
	}
	for _, m := range msgs {
		fmt.Println(m)
	}
	return nil
}

var pipelineDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS pipeline",
	Ctx: []cli.Arg{
		{Name: "project-key"},
	},
	Args: []cli.Arg{
		{Name: "pipeline-name"},
	},
}

func pipelineDeleteRun(v cli.Values) error {
	err := client.PipelineDelete(v["project-key"], v["pipeline-name"])
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrPipelineNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}

	return err
}
