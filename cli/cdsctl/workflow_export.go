package main

import (
	"fmt"
	"net/http"
	"reflect"

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
	mods := []cdsclient.RequestModifier{
		func(r *http.Request) {
			q := r.URL.Query()
			q.Set("format", c.GetString("format"))
			r.URL.RawQuery = q.Encode()
		},
	}
	if c.GetBool("with-permissions") {
		mods = append(mods,
			func(r *http.Request) {
				q := r.URL.Query()
				q.Set("withPermissions", "true")
				r.URL.RawQuery = q.Encode()
			},
		)
	}

	btes, err := client.WorkflowExport(c.GetString(_ProjectKey), c.GetString(_WorkflowName), mods...)
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}
