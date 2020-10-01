package main

import (
	"context"
	"encoding/json"
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
		{Name: "branch", Type: cli.FlagString},
		{Name: "message", Type: cli.FlagString},
	},
}

func workflowTransformAsCodeRun(v cli.Values) (interface{}, error) {
	projectKey := v.GetString(_ProjectKey)

	w, err := client.WorkflowGet(projectKey, v.GetString(_WorkflowName))
	if err != nil {
		return nil, err
	}
	if w.FromRepository != "" {
		return nil, sdk.ErrWorkflowAlreadyAsCode
	}

	noInteractive := v.GetBool("no-interactive")

	branch := v.GetString("branch")
	message := v.GetString("message")
	if !noInteractive && branch == "" {
		branch = cli.AskValue("Give a branch name")
	}
	if !noInteractive && message == "" {
		message = cli.AskValue("Give a commit message")
	}

	ctx := context.Background()
	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)
	chanErrorReceived := make(chan error)

	sdk.NewGoRoutines().Run(ctx, "WebsocketEventsListenCmd", func(ctx context.Context) {
		client.WebsocketEventsListen(ctx, sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived, chanErrorReceived)
	})

	ope, err := client.WorkflowTransformAsCode(projectKey, v.GetString(_WorkflowName), branch, message)
	if err != nil {
		return nil, err
	}

	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type:          sdk.WebsocketFilterTypeOperation,
		ProjectKey:    projectKey,
		OperationUUID: ope.UUID,
	}}

	if !v.GetBool("silent") {
		fmt.Println("CDS is pushing files on your repository. A pull request will be created, please wait...")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
forLoop:
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting operation to complete")
		case err := <-chanErrorReceived:
			fmt.Printf("Error: %v\n", err)
		case evt := <-chanMessageReceived:
			if evt.Event.EventType == fmt.Sprintf("%T", sdk.EventOperation{}) {
				if err := json.Unmarshal(evt.Event.Payload, &ope); err != nil {
					return nil, fmt.Errorf("cannot parse operation from received event: %v", err)
				}
				if ope.Status > sdk.OperationStatusProcessing {
					break forLoop
				}
			}
		}
	}

	if ope.Status == sdk.OperationStatusError {
		sdk.Exit("An error occured when migrate: %v", ope.Error)
	}

	urlSplitted := strings.Split(ope.Setup.Push.PRLink, "/")
	id, err := strconv.Atoi(urlSplitted[len(urlSplitted)-1])
	if err != nil {
		return nil, fmt.Errorf("cannot read id from pull request URL %s: %v", ope.Setup.Push.PRLink, err)
	}
	response := struct {
		URL string `cli:"url" json:"url"`
		ID  int    `cli:"id" json:"id"`
	}{
		URL: ope.Setup.Push.PRLink,
		ID:  id,
	}
	switch ope.Status {
	case sdk.OperationStatusError:
		return nil, fmt.Errorf("cannot perform operation: %v", ope.Error)
	}
	return response, nil
}
