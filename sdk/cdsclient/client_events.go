package cdsclient

import (
	"context"
	"log"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) WebsocketEventsListen(ctx context.Context, goRoutines *sdk.GoRoutines, chanMsgToSend <-chan []sdk.WebsocketFilter, chanMsgReceived chan<- sdk.WebsocketEvent) {
	for ctx.Err() == nil {
		if err := c.RequestWebsocket(ctx, goRoutines, "/ws", chanMsgToSend, chanMsgReceived); err != nil {
			log.Printf("websocket error: %v\n", err)
		}
		time.Sleep(1 * time.Second)
	}
}
