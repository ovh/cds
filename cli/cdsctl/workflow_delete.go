package main

import (
	"fmt"
	"os"

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
}

func workflowDeleteRun(v cli.Values) error {
	err := client.WorkflowDelete(v[_ProjectKey], v[_WorkflowName])
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrWorkflowNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	return err
}
