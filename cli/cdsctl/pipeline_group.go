package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
)

var pipelineGroupCmd = cli.Command{
	Name:  "group",
	Short: "Manage CDS group linked to a pipeline",
}

func pipelineGroup() *cobra.Command {
	return cli.NewCommand(pipelineGroupCmd, nil, []*cobra.Command{
		cli.NewCommand(pipelineGroupImportCmd, pipelineGroupImportRun, nil, withAllCommandModifiers()...),
	})
}

var pipelineGroupImportCmd = cli.Command{
	Name:  "import",
	Short: "Import group linked to a CDS pipeline",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "pipeline-name"},
		{Name: "path"},
	},
	Flags: []cli.Flag{
		{
			Name:  "force",
			Usage: "Use force flag to replace groups in your pipeline",
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

func pipelineGroupImportRun(v cli.Values) error {
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

	if _, err := client.PipelineGroupsImport(v.GetString(_ProjectKey), v.GetString("pipeline-name"), reader, format, v.GetBool("force")); err != nil {
		return err
	}
	fmt.Printf("Groups imported in pipeline %s with success\n", v.GetString("pipeline-name"))

	return nil
}
