package main

import (
	"fmt"
	"reflect"

	"github.com/ovh/cds/cli"
)

var workflowExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a workflow",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
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
	btes, err := client.WorkflowExport(c.GetString(_ProjectKey), c.GetString(_WorkflowName), c.GetBool("with-permissions"), c.GetString("format"))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}
