package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/ovh/cds/cli"
)

var workflowImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a workflow",
	Long: `
		In case you want to import just your workflow.
		
		If you want to update also dependencies likes pipelines, applications or environments at same time you have to use workflow push instead workflow import.
	`,
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "filename"},
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
	path := c.GetString("filename")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var format = "yaml"
	if strings.HasSuffix(path, ".json") {
		format = "json"
	}

	msgs, err := client.WorkflowImport(c.GetString("project-key"), f, format, c.GetBool("force"))
	if err != nil {
		return err
	}

	for _, s := range msgs {
		fmt.Println(s)
	}

	return nil
}
