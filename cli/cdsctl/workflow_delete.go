package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var workflowDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS workflow",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Flags: []cli.Flag{
		{
			Name:      "with-dependencies",
			ShortHand: "d",
			Usage:     "delete and clean workflow dependencies",
			Type:      cli.FlagBool,
		},
	},
}

func workflowDeleteRun(v cli.Values) error {
	mod := func(r *http.Request) {
		q := r.URL.Query()
		b := v.GetBool("with-dependencies")
		q.Set("withDependencies", strconv.FormatBool(b))
		r.URL.RawQuery = q.Encode()
	}
	err := client.WorkflowDelete(v.GetString(_ProjectKey), v.GetString(_WorkflowName), mod)
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	return err
}
