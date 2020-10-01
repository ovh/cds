package cdsclient

import (
	"context"
	"encoding/json"
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
					chanErrorReceived <- sdk.WrapError(err, "unable to marshal message: %s", string(m))
					continue
				}
				chanMsgToSend <- m
			case m := <-chanMsgReceived:
				var wsEvent sdk.WebsocketEvent
				if err := json.Unmarshal(m, &wsEvent); err != nil {
					chanErrorReceived <- sdk.WrapError(err, "unable to unmarshal message: %s", string(m))
					continue
				}
				chanEventReceived <- wsEvent
			}
		}
	})

	for ctx.Err() == nil {
		if err := c.RequestWebsocket(ctx, goRoutines, "/ws", chanMsgToSend, chanMsgReceived, chanErrorReceived); err != nil {
			chanErrorReceived <- sdk.WrapError(err, "websocket error")
		}
		time.Sleep(1 * time.Second)
	}
}
