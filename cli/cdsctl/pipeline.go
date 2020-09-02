package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

var pipelineCmd = cli.Command{
	Name:    "pipeline",
	Aliases: []string{"pipelines"},
	Short:   "Manage CDS pipeline",
}

func pipeline() *cobra.Command {
	return cli.NewCommand(pipelineCmd, nil, []*cobra.Command{
		cli.NewListCommand(pipelineListCmd, pipelineListRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(pipelineUsageCmd, pipelineUsageRun, nil, withAllCommandModifiers()...),
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

var pipelineUsageCmd = cli.Command{
	Name:  "usage",
	Short: "CDS pipeline usage",
	Long:  "PATH: Path or URL of pipeline",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "pipeline-name"},
	},
}

func pipelineUsageRun(v cli.Values) (cli.ListResult, error) {
	pipeline, err := client.PipelineGet(v.GetString(_ProjectKey), v.GetString("pipeline-name"), cdsclient.WithWorkflows())
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(pipeline.Usage.Workflows), nil
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
			Type:    cli.FlagString,
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func pipelineExportRun(v cli.Values) error {
	btes, err := client.PipelineExport(v.GetString(_ProjectKey), v.GetString("pipeline-name"),
		cdsclient.Format(v.GetString("format")))
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
	path := v.GetString("path")

	var reader io.ReadCloser
	var err error
	if sdk.IsURL(path) {
		reader, err = exportentities.OpenURL(path)
	} else {
		reader, err = exportentities.OpenFile(path)
	}
	if err != nil {
		return err
	}
	defer reader.Close() // nolint

	format, err := exportentities.GetFormatFromPath(path)
	if err != nil {
		return err
	}

	mods := []cdsclient.RequestModifier{
		cdsclient.ContentType(format.ContentType()),
	}
	if v.GetBool("force") {
		mods = append(mods, cdsclient.Force())
	}

	msgs, err := client.PipelineImport(v.GetString(_ProjectKey), reader, mods...)
	for _, m := range msgs {
		fmt.Println(m)
	}
	return err
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
