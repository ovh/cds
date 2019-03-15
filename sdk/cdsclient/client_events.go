package cdsclient

import (
	"context"
	"log"
	"time"
)

func (c *client) EventsListen(ctx context.Context, chanSSEvt chan<- SSEvent) {
	for ctx.Err() == nil {
		if err := c.RequestSSEGet(ctx, "/events", chanSSEvt); err != nil {
			log.Println("QueuePolling", err)
		}
		time.Sleep(1 * time.Second)
	}
}
