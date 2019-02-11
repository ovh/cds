package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var pipelineCmd = cli.Command{
	Name:  "pipeline",
	Short: "Manage CDS pipeline",
}

func pipeline() *cobra.Command {
	return cli.NewCommand(pipelineCmd, nil, []*cobra.Command{
		cli.NewListCommand(pipelineListCmd, pipelineListRun, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(pipelineDeleteCmd, pipelineDeleteRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(pipelineExportCmd, pipelineExportRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(pipelineImportCmd, pipelineImportRun, nil, withAllCommandModifiers()...),
	})
}

var pipelineListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS pipelines",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func pipelineListRun(v cli.Values) (cli.ListResult, error) {
	pipelines, err := client.PipelineList(v.GetString(_ProjectKey))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(pipelines), nil
}

var pipelineExportCmd = cli.Command{
	Name:  "export",
	Short: "Export CDS pipeline",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
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
			Default: "yml",
		},
	},
}

func pipelineExportRun(v cli.Values) error {
	btes, err := client.PipelineExport(v.GetString(_ProjectKey), v.GetString("pipeline-name"), v.GetString("format"))
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
		{Name: _ProjectKey},
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
			Type:    cli.FlagBool,
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

	msgs, err := client.PipelineImport(v.GetString(_ProjectKey), reader, format, v.GetBool("force"))
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
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "pipeline-name"},
	},
}

func pipelineDeleteRun(v cli.Values) error {
	err := client.PipelineDelete(v.GetString(_ProjectKey), v.GetString("pipeline-name"))
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrPipelineNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}

	return err
}
