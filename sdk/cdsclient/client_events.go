package cdsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) WebsocketEventsListen(ctx context.Context, goRoutines *sdk.GoRoutines, chanFilterToSend <-chan []sdk.WebsocketFilter, chanEventReceived chan<- sdk.WebsocketEvent, chanErrorReceived chan<- error) {
	chanMsgReceived := make(chan json.RawMessage)
	chanMsgToSend := make(chan json.RawMessage)

	goRoutines.Exec(ctx, "WebsocketEventsListen", func(ctx context.Context) {
		for {
			select {
			case f := <-chanFilterToSend:
				m, err := json.Marshal(f)
				if err != nil {
					chanErrorReceived <- newError(fmt.Errorf("unable to marshal message: %s: %v", string(m), err))
					continue
				}
				chanMsgToSend <- m
			case m := <-chanMsgReceived:
				var wsEvent sdk.WebsocketEvent
				if err := sdk.JSONUnmarshal(m, &wsEvent); err != nil {
					chanErrorReceived <- newError(fmt.Errorf("unable to unmarshal message: %s: %v", string(m), err))
					continue
				}
				chanEventReceived <- wsEvent
			}
		}
	})

	for ctx.Err() == nil {
		if err := c.RequestWebsocket(ctx, goRoutines, "/ws", chanMsgToSend, chanMsgReceived, chanErrorReceived); err != nil {
			chanErrorReceived <- newError(fmt.Errorf("websocket error: %v", err))
		}
		time.Sleep(1 * time.Second)
	}
}
