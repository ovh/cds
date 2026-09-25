package cdn

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

// A shutdown must reach the connections already accepted. Before this, only the listener was
// closed and a reader blocked on an idle connection stayed there until the process died, while
// its client kept writing into a socket nobody read any more.
func TestConnRegistry_CloseAllReachesTheClient(t *testing.T) {
	r := newConnRegistry()
	require.Equal(t, 0, r.len())

	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	r.add(server)
	require.Equal(t, 1, r.len())

	require.Equal(t, 1, r.closeAll())
	require.Equal(t, 0, r.len(), "a closed connection is dropped, so its own deferred close does not find it")

	// The client must learn about it rather than stay blocked.
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	_, err := client.Read(make([]byte, 1))
	require.Error(t, err, "the client side of a closed connection must not keep waiting")

	// Removing a connection that is already gone is what every reader does on its way out.
	r.remove(server)
	require.Equal(t, 0, r.len())
	require.Equal(t, 0, r.closeAll())
}

// The rate limiter carries the context of the service. Once it is cancelled it refuses every
// call instantly, so a read loop that retries on that error spins: it burns a core and floods
// the logs for the whole termination grace, on every open connection, at every deploy.
func TestReadTCPMessages_ReturnsOnShutdownInsteadOfSpinning(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctx, cancel := context.WithCancel(context.TODO())
	globalRateLimit = NewRateLimiter(ctx, 1024*1024*1024, 1024)

	s := Service{}
	s.Cfg.Log.StepMaxSize = 1024
	s.Cfg.Log.StepLinesRateLimit = 1000

	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.readTCPMessages(ctx, server, func(context.Context, []byte) error { return nil })
	}()

	// Nothing is ever written: the reader is idle, exactly like a worker between two steps.
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("readTCPMessages did not return after the service context was cancelled: it is spinning")
	}
}
