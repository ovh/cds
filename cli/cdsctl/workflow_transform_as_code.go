package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var workflowTransformAsCodeCmd = cli.Command{
	Name:  "ascode",
	Short: "Transform an existing workflow to an as code workflow",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Flags: []cli.Flag{
		{Name: "silent", Type: cli.FlagBool},
	},
}

func workflowTransformAsCodeRun(v cli.Values) (interface{}, error) {
	w, err := client.WorkflowGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName))
	if err != nil {
		return nil, err
	}
	if w.FromRepository != "" {
		return nil, sdk.ErrWorkflowAlreadyAsCode
	}

	ope, err := client.WorkflowTransformAsCode(v.GetString(_ProjectKey), v.GetString(_WorkflowName))
	if err != nil {
		return nil, err
	}

	if !v.GetBool("silent") {
		fmt.Println("CDS is pushing files on your repository. A pull request will be created, please wait...")
	}
	for {
		if err := client.WorkflowTransformAsCodeFollow(v.GetString(_ProjectKey), v.GetString(_WorkflowName), ope); err != nil {
			return nil, err
		}
		if ope.Status > sdk.OperationStatusProcessing {
			break
		}
		time.Sleep(1 * time.Second)
	}

	urlSplitted := strings.Split(ope.Setup.Push.PRLink, "/")
	id, err := strconv.Atoi(urlSplitted[len(urlSplitted)-1])
	if err != nil {
		return nil, fmt.Errorf("cannot read id from pull request URL %s: %v", ope.Setup.Push.PRLink, err)
	}
	reponse := struct {
		URL string `cli:"url" json:"url"`
		ID  int    `cli:"id" json:"id"`
	}{
		URL: ope.Setup.Push.PRLink,
		ID:  id,
	}
	switch ope.Status {
	case sdk.OperationStatusError:
		return nil, fmt.Errorf("cannot perform operation: %s", ope.Error)
	}
	return reponse, nil
}
