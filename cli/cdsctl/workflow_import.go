package main

import (
	"fmt"
	"reflect"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
)

var workflowImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a workflow",
	Long: `
In case you want to import just your workflow. Instead of use a local file you can also use an URL to your yaml file.
		
If you want to update also dependencies likes pipelines, applications or environments at same time you have to use workflow push instead workflow import.

	`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "path"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.Bool,
			Name:    "force",
			Usage:   "Override workflow if exists",
			Default: "false",
		},
	},
}

func workflowImportRun(c cli.Values) error {
	path := c.GetString("path")
	contentFile, format, err := exportentities.OpenPath(path)
	if err != nil {
		return err
	}
	defer contentFile.Close() //nolint
	formatStr, _ := exportentities.GetFormatStr(format)

	msgs, err := client.WorkflowImport(c.GetString(_ProjectKey), contentFile, formatStr, c.GetBool("force"))
	if err != nil {
		return err
	}

	for _, s := range msgs {
		fmt.Println(s)
	}

	return nil
}
