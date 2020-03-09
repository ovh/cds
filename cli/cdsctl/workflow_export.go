package main

import (
	"fmt"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
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
			Type:    cli.FlagBool,
			Name:    "with-permissions",
			Usage:   "Export permissions",
			Default: "false",
		},
		{
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func workflowExportRun(c cli.Values) error {
	mods := []cdsclient.RequestModifier{cdsclient.Format(c.GetString("format"))}
	if c.GetBool("with-permissions") {
		mods = append(mods, cdsclient.WithPermissions())
	}
	buf, err := client.WorkflowExport(c.GetString(_ProjectKey), c.GetString(_WorkflowName), mods...)
	if err != nil {
		return err
	}
	fmt.Println(string(buf))
	return nil
}
