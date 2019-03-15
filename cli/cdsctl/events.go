package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
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
}

func eventsListenRun(v cli.Values) error {
	ctx := context.Background()
	chanSSE := make(chan cdsclient.SSEvent)

	sdk.GoRoutine(ctx, "EventsListenCmd", func(ctx context.Context) {
		client.EventsListen(ctx, chanSSE)
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt := <-chanSSE:
			var e sdk.Event
			content, _ := ioutil.ReadAll(evt.Data)
			_ = json.Unmarshal(content, &e)
			if e.EventType == "" {
				continue
			}
			fmt.Printf("%s: %s %s %s\n", e.EventType, e.ProjectKey, e.WorkflowName, e.Status)
		}
	}
}
