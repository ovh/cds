package websocket

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

type fakePubSub struct {
	messages chan string
}

func (f *fakePubSub) Unsubscribe(_ context.Context, _ ...string) error { return nil }

func (f *fakePubSub) GetMessage(ctx context.Context) (string, error) {
	select {
	case m := <-f.messages:
		return m, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

type flakyPubSub struct {
	remainingFailures int32
	messages          chan string
}

func (f *flakyPubSub) Unsubscribe(_ context.Context, _ ...string) error { return nil }

func (f *flakyPubSub) GetMessage(ctx context.Context) (string, error) {
	if atomic.AddInt32(&f.remainingFailures, -1) >= 0 {
		return "", errors.New("pubsub unavailable")
	}
	select {
	case m := <-f.messages:
		return m, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Without the option the broker reads one message per defaultReadInterval, so 500 pending
// messages take 25 seconds. The API and hatchery brokers rely on that pacing.
func TestBroker_throttlesReadsByDefault(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const nbMessages = 500
	ps := &fakePubSub{messages: make(chan string, nbMessages)}
	for i := 0; i < nbMessages; i++ {
		ps.messages <- "message"
	}

	var received int64
	b := NewBroker()
	b.OnMessage(func(_ []byte) { atomic.AddInt64(&received, 1) })
	b.Init(ctx, sdk.NewGoRoutines(ctx), ps)

	time.Sleep(250 * time.Millisecond)
	require.Less(t, atomic.LoadInt64(&received), int64(nbMessages/10), "broker forwarded without pacing its reads")
}

func TestBroker_withoutReadThrottlingForwardsEverythingPending(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const nbMessages = 500
	ps := &fakePubSub{messages: make(chan string, nbMessages)}
	for i := 0; i < nbMessages; i++ {
		ps.messages <- "message"
	}

	var received int64
	b := NewBroker(WithoutReadThrottling())
	b.OnMessage(func(_ []byte) { atomic.AddInt64(&received, 1) })
	b.Init(ctx, sdk.NewGoRoutines(ctx), ps)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&received) >= nbMessages
	}, 5*time.Second, 10*time.Millisecond, "broker did not forward the %d pending messages", nbMessages)
}

// Init starts subscribe with Run and not RunWithRestart, so a loop that returned on a
// recoverable error would leave the process without a broker until it is restarted.
func TestBroker_survivesPubSubErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ps := &flakyPubSub{remainingFailures: 3, messages: make(chan string, 1)}
	ps.messages <- "message"

	var received int64
	b := NewBroker()
	b.OnMessage(func(_ []byte) { atomic.AddInt64(&received, 1) })
	b.Init(ctx, sdk.NewGoRoutines(ctx), ps)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&received) == 1
	}, 5*time.Second, 10*time.Millisecond, "broker stopped forwarding after a pubsub error")
}

// The send on the broker channel has no reader once the consumer is gone, so it must not
// outlive the context.
func TestBroker_subscribeReturnsOnCancelWhileBlockedOnSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ps := &fakePubSub{messages: make(chan string, 1)}
	ps.messages <- "message"

	b := NewBroker()
	done := make(chan struct{})
	go func() {
		b.subscribe(ctx, ps)
		close(done)
	}()

	time.Sleep(250 * time.Millisecond) // let subscribe tick, read, then block on the send
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("subscribe did not return after the context was cancelled")
	}
}
