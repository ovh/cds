package websocket

import (
	"context"
	"io"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func NewBroker() *Broker {
	return &Broker{chanMessages: make(chan []byte)}
}

type Broker struct {
	chanMessages chan []byte
	onMessage    func(m []byte)
}

func (b *Broker) OnMessage(f func(m []byte)) { b.onMessage = f }

//Init the websocketBroker
func (b *Broker) Init(ctx context.Context, gorts *sdk.GoRoutines, pubSub cache.PubSub, panicCallback ...func(s string) (io.WriteCloser, error)) {
	// Start cache Subscription
	gorts.Run(ctx, "websocket.Broker.Init.cacheSubscribe", func(ctx context.Context) {
		b.subscribe(ctx, pubSub)
	}, panicCallback...)

	gorts.Run(ctx, "websocket.Broker.Init.start", func(ctx context.Context) {
		b.start(ctx, gorts, panicCallback...)
	}, panicCallback...)
}

// Start the broker
func (b *Broker) start(ctx context.Context, gorts *sdk.GoRoutines, panicCallback ...func(s string) (io.WriteCloser, error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-b.chanMessages:
			if b.onMessage != nil {
				b.onMessage(msg)
			}
		}
	}
}

func (b *Broker) subscribe(ctx context.Context, pubSub cache.PubSub) {
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if ctx.Err() != nil {
				continue
			}
			msg, err := pubSub.GetMessage(ctx)
			if err != nil {
				log.Warning(ctx, "websocket.Broker> cannot get message from pubsub %s: %s", msg, err)
				continue
			}
			b.chanMessages <- []byte(msg)
		}
	}
}
