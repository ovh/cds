package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var eventsCmd = cli.Command{
	Name:  "events",
	Short: "Listen CDS Events",
}

func events() *cobra.Command {
	return cli.NewCommand(eventsCmd, nil, []*cobra.Command{
		cli.NewCommand(eventsListenCmd, eventsListenRun, nil, withAllCommandModifiers()...),
	})
}

var eventsListenCmd = cli.Command{
	Name:  "listen",
	Short: "Listen CDS events",
	Flags: []cli.Flag{
		{
			Name:  "project",
			Usage: "project key to listen",
			Type:  cli.FlagString,
		},
		{
			Name:  "workflow",
			Usage: "workflow name to listen",
			Type:  cli.FlagString,
		},
	},
}

func eventsListenRun(v cli.Values) error {
	ctx := context.Background()
	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)

	sdk.GoRoutine(ctx, "WebsocketEventsListenCmd", func(ctx context.Context) {
		client.WebsocketEventsListen(ctx, chanMessageToSend, chanMessageReceived)
	})

	var t sdk.WebsocketFilterType
	switch {
	case v.GetString("workflow") != "":
		t = sdk.WebsocketFilterTypeWorkflow
	default:
		t = sdk.WebsocketFilterTypeProject
	}
	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type:         t,
		ProjectKey:   v.GetString("project"),
		WorkflowName: v.GetString("workflow"),
	}}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt := <-chanMessageReceived:
			if evt.Event.EventType == "" {
				continue
			}
			fmt.Printf("%s: %s %s %s\n", evt.Event.EventType, evt.Event.ProjectKey, evt.Event.WorkflowName, evt.Event.Status)
		}
	}
}
