package main

import (
	"fmt"
	"reflect"

	"github.com/ovh/cds/cli"
)

var workflowExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.Bool,
			Name:    "with-permissions",
			Usage:   "Export permissions",
			Default: "false",
		},
		{
			Kind:    reflect.String,
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func workflowExportRun(c cli.Values) error {
	btes, err := client.WorkflowExport(c.GetString("project-key"), c.GetString("workflow-name"), c.GetBool("with-permissions"), c.GetString("format"))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}
