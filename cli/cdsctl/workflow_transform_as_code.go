package main

import (
	"fmt"
	"time"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var workflowTransformAsCodeCmd = cli.Command{
	Name:  "ascode",
	Short: "Transform an existing workflow to an as code workflow",
	Args: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
}

func workflowTransformAsCodeRun(v cli.Values) error {
	w, err := client.WorkflowGet(v[_ProjectKey], v[_WorkflowName])
	if err != nil {
		return err
	}
	if w.FromRepository != "" {
		fmt.Printf("Workflow is already as code. \n")
		return nil
	}

	ope, err := client.WorkflowTransformAsCode(v[_ProjectKey], v[_WorkflowName])
	if err != nil {
		return err
	}

	fmt.Printf("CDS is pushing files on your repository. A pull request will be created, please wait...\n")
	for {
		if err := client.WorkflowTransformAsCodeFollow(v[_ProjectKey], v[_WorkflowName], ope); err != nil {
			return err
		}
		if ope.Status > sdk.OperationStatusProcessing {
			break
		}
		time.Sleep(1 * time.Second)
	}
	switch ope.Status {
	case sdk.OperationStatusDone:
		fmt.Printf(cli.Blue("%s\n", ope.Setup.Push.PRLink))
	case sdk.OperationStatusError:
		sdk.Exit("An error occured: %s\n", ope.Error)
	}
	return nil
}
