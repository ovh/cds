package websocket

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

const (
	// defaultReadInterval paces the pubsub reads: at most one message per interval, whatever
	// the rate the publishers produce.
	defaultReadInterval = 50 * time.Millisecond
	errorBackoff        = 50 * time.Millisecond
)

// BrokerOption configures a broker at construction.
type BrokerOption func(*Broker)

// WithoutReadThrottling lets the broker forward messages as fast as the pubsub delivers them,
// instead of one per defaultReadInterval. The handler then runs at the rate the publishers
// produce, so this suits a broker whose handler is cheap and bounded.
func WithoutReadThrottling() BrokerOption {
	return func(b *Broker) { b.readInterval = 0 }
}

func NewBroker(opts ...BrokerOption) *Broker {
	b := &Broker{
		chanMessages: make(chan []byte),
		readInterval: defaultReadInterval,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

type Broker struct {
	chanMessages chan []byte
	onMessage    func(m []byte)
	readInterval time.Duration
}

func (b *Broker) OnMessage(f func(m []byte)) { b.onMessage = f }

// Init the websocketBroker
func (b *Broker) Init(ctx context.Context, gorts *sdk.GoRoutines, pubSub cache.PubSub) {
	// Start cache Subscription
	gorts.Run(ctx, "websocket.Broker.Init.cacheSubscribe", func(ctx context.Context) {
		b.subscribe(ctx, pubSub)
	})

	gorts.Run(ctx, "websocket.Broker.Init.start", func(ctx context.Context) {
		b.start(ctx)
	})
}

// Start the broker
func (b *Broker) start(ctx context.Context) {
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

// subscribe forwards the messages of the pubsub to the broker channel. GetMessage blocks until
// a message arrives, so the ticker never protects this loop from spinning: it caps the broker
// at one message per interval, which is a throttle on the handler and nothing else. The backoff
// on the error path is what covers an implementation that fails without blocking.
func (b *Broker) subscribe(ctx context.Context, pubSub cache.PubSub) {
	var tick *time.Ticker
	if b.readInterval > 0 {
		tick = time.NewTicker(b.readInterval)
		defer tick.Stop()
	}

	for {
		if ctx.Err() != nil {
			return
		}

		if tick != nil {
			select {
			case <-tick.C:
			case <-ctx.Done():
				return
			}
		}

		msg, err := pubSub.GetMessage(ctx)
		if err != nil {
			// Returning here ends the broker for the life of the process: Init starts this
			// with Run, not RunWithRestart. So only a cancelled context leaves the loop,
			// every other error is retried.
			if ctx.Err() != nil {
				return
			}
			log.Warn(ctx, "websocket.Broker> cannot get message from pubsub: %v", err)
			select {
			case <-time.After(errorBackoff):
			case <-ctx.Done():
				return
			}
			continue
		}

		select {
		case b.chanMessages <- []byte(msg):
		case <-ctx.Done():
			return
		}
	}
}
