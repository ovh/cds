package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tevino/abool"
)

func TestCommonClient_Send_RefusesToQueueBehindAClientThatFellBehind(t *testing.T) {
	c := &CommonClient{uuid: "client", isClosed: abool.NewBool(false)}

	// Stands in for a write in progress on a client that stopped reading: every Send below queues
	// behind it, as they would behind a write waiting on the socket.
	c.mutex.Lock()

	for i := 0; i < maxPendingWrites; i++ {
		go func() { _ = c.Send("message") }()
	}
	require.Eventually(t, func() bool { return c.pending.Load() == maxPendingWrites }, 5*time.Second, time.Millisecond,
		"the writes of a client that does not read should be waiting their turn")

	err := c.Send("one too many")
	require.Error(t, err, "a write past the bound is refused rather than held")
	require.Contains(t, err.Error(), "too far behind")
	require.Equal(t, int64(maxPendingWrites), c.pending.Load(), "a refused write does not count as pending")

	c.mutex.Unlock()
	require.Eventually(t, func() bool { return c.pending.Load() == 0 }, 5*time.Second, time.Millisecond,
		"the writes waiting are released once the client is written to again")
}
