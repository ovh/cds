package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var eventsCmd = cli.Command{
	Name:    "events",
	Aliases: []string{"event"},
	Short:   "Listen CDS Events",
}

func events() *cobra.Command {
	return cli.NewCommand(eventsCmd, nil, []*cobra.Command{
		cli.NewCommand(eventsListenCmd, eventsListenRun, nil, withAllCommandModifiers()...),
	})
}

var eventsListenCmd = cli.Command{
	Name:  "listen",
	Short: "Listen CDS events",
	Example: `  cdsctl events listen --queue
  cdsctl events listen --global
  cdsctl events listen --project MYPROJ
  cdsctl events listen --project MYPROJ --workflow my-workflow
  `,
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
		{
			Name:  "queue",
			Usage: "listen job queue events",
			Type:  cli.FlagBool,
		},
		{
			Name:  "global",
			Usage: "listen global events",
			Type:  cli.FlagBool,
		},
	},
}

func eventsListenRun(v cli.Values) error {
	ctx := context.Background()
	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)
	chanErrorReceived := make(chan error)

	sdk.NewGoRoutines().Run(ctx, "WebsocketEventsListenCmd", func(ctx context.Context) {
		client.WebsocketEventsListen(ctx, sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived, chanErrorReceived)
	})

	switch {
	case v.GetString("project") != "" && v.GetString("workflow") != "":
		chanMessageToSend <- []sdk.WebsocketFilter{{
			Type:         sdk.WebsocketFilterTypeWorkflow,
			ProjectKey:   v.GetString("project"),
			WorkflowName: v.GetString("workflow"),
		}}
	case v.GetString("project") != "":
		chanMessageToSend <- []sdk.WebsocketFilter{{
			Type:       sdk.WebsocketFilterTypeProject,
			ProjectKey: v.GetString("project"),
		}}
	case v.GetBool("queue"):
		chanMessageToSend <- []sdk.WebsocketFilter{{
			Type: sdk.WebsocketFilterTypeQueue,
		}}
	case v.GetBool("global"):
		chanMessageToSend <- []sdk.WebsocketFilter{{
			Type: sdk.WebsocketFilterTypeGlobal,
		}}
	default:
		return fmt.Errorf("invalid given parameters")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-chanErrorReceived:
			fmt.Printf("Error: %v\n", err)
		case evt := <-chanMessageReceived:
			if evt.Event.EventType == "" {
				continue
			}
			fmt.Printf("%s: %s %s %s\n", evt.Event.EventType, evt.Event.ProjectKey, evt.Event.WorkflowName, evt.Event.Status)
		}
	}
}
