package cdn

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk/log/hook/graylog"
)

// readAckFrame reads one NUL terminated ack from the client side of a pipe, from another
// goroutine: net.Pipe is synchronous, so the reader must already be waiting when flush writes.
// The ack state itself stays in the test goroutine, as it is in the intake: one connection, one
// goroutine, no lock.
func readAckFrame(conn net.Conn) <-chan string {
	ch := make(chan string, 1)
	go func() {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 64)
		n, err := conn.Read(buf)
		if err != nil || n < 2 || buf[n-1] != 0 {
			ch <- ""
			return
		}
		ch <- string(buf[:n-1])
	}()
	return ch
}

// The sequence must be read without unmarshalling the message again: the intake already pays
// for one parse per line.
func TestExtractAckSeq(t *testing.T) {
	seq, ok := extractAckSeq([]byte(`{"version":"1.1","_ack_seq":42}`))
	require.True(t, ok)
	require.Equal(t, uint64(42), seq)

	// A message from a worker that does not know the protocol
	_, ok = extractAckSeq([]byte(`{"version":"1.1","short_message":"hello"}`))
	require.False(t, ok, "an old worker must not enable the acks")

	_, ok = extractAckSeq([]byte(`{"_ack_seq":0}`))
	require.False(t, ok, "sequence numbers are 1 based, 0 means no sequence")

	_, ok = extractAckSeq([]byte(`{"_ack_seq":"nope"}`))
	require.False(t, ok)

	_, ok = extractAckSeq([]byte(`{"_ack_seq":99999999999999999999999}`))
	require.False(t, ok, "an absurd sequence number must not overflow the parser")
}

// The ack carries the highest CONTIGUOUS sequence: a hole must stop the acks from progressing,
// otherwise the worker would release a message that never arrived.
func TestAckState_ContiguousStopsOnHole(t *testing.T) {
	a := newAckState()

	a.observe(1)
	a.observe(2)
	require.Equal(t, uint64(2), a.contiguous)

	// 3 is missing
	a.observe(4)
	a.observe(5)
	require.Equal(t, uint64(2), a.contiguous, "the worker must keep 3, 4 and 5 replayable")

	// the hole is filled: everything that was waiting behind it is acknowledged at once
	a.observe(3)
	require.Equal(t, uint64(5), a.contiguous)
	require.Empty(t, a.outOfOrder)
}

// A replayed line is a line we had already acknowledged: it must not move anything backwards.
func TestAckState_ReplayedSequenceIsIgnored(t *testing.T) {
	a := newAckState()
	a.observe(1)
	a.observe(2)
	a.observe(1)
	require.Equal(t, uint64(2), a.contiguous)
}

// Cadence: one ack every N messages, so that a busy connection does not pay a syscall per line.
func TestAckState_FlushesEveryNMessages(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	server, client := net.Pipe()
	t.Cleanup(func() { _ = server.Close(); _ = client.Close() })

	a := newAckState()
	a.everyMessages = 3
	a.everyDuration = time.Hour // only the message count may trigger it here

	ctx := context.TODO()
	a.observe(1)
	a.flush(ctx, server)
	a.observe(2)
	a.flush(ctx, server)
	require.False(t, a.due(), "not yet: only 2 messages")

	a.observe(3)
	ack := readAckFrame(client)
	a.flush(ctx, server)
	require.Equal(t, `{"ack":3}`, <-ack, "the ack must use the same NUL framing as the gelf messages")
	require.False(t, a.pending)
	require.Equal(t, 0, a.sinceLastAck)
}

// Cadence: and at least one ack every T, so that a slow connection still gets its confirmation
// and the worker's replay ring is released.
func TestAckState_FlushesAfterDuration(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	server, client := net.Pipe()
	t.Cleanup(func() { _ = server.Close(); _ = client.Close() })

	a := newAckState()
	a.everyMessages = 1000 // out of reach
	a.everyDuration = 20 * time.Millisecond

	a.observe(1)
	require.False(t, a.due(), "the duration has not elapsed yet")

	time.Sleep(30 * time.Millisecond)
	require.True(t, a.due())

	ack := readAckFrame(client)
	a.flush(context.TODO(), server)
	require.Equal(t, `{"ack":1}`, <-ack)
}

// A client that never reads must not be able to block the goroutine that reads its logs: the
// write deadline fires, the ack is skipped and the intake goes on.
func TestAckState_WriteDeadlineSkipsTheAck(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	server, client := net.Pipe()
	t.Cleanup(func() { _ = server.Close(); _ = client.Close() })
	// nobody ever reads on the client side: net.Pipe is synchronous, so the write blocks

	a := newAckState()
	a.everyMessages = 1
	a.writeTimeout = 20 * time.Millisecond

	a.observe(1)

	start := time.Now()
	a.flush(context.TODO(), server)
	elapsed := time.Since(start)

	require.Less(t, elapsed, time.Second, "flush must give up on the write deadline")
	require.Equal(t, 1, a.writeFailures)
	require.True(t, a.pending, "the ack is still due: acks are cumulative, the next one says the same")

	// And the connection is still usable for the next ack
	a.observe(2)
	ack := readAckFrame(client)
	a.flush(context.TODO(), server)
	require.Equal(t, `{"ack":2}`, <-ack)
}

// serveAcks runs the acknowledgement part of handleConnection, with the very same gates: the
// state comes from the feature flag, and everything the protocol does is conditioned on it.
func serveAcks(s *Service, ln net.Listener) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close() // nolint

	acks := s.newConnectionAckState()
	if acks != nil {
		acks.everyMessages = 1
	}

	buf := make([]byte, 1024)
	var frame []byte
	for {
		if acks.active() {
			_ = conn.SetReadDeadline(time.Now().Add(acks.everyDuration))
		}
		n, err := conn.Read(buf)
		for i := 0; i < n; i++ {
			if buf[i] != 0 {
				frame = append(frame, buf[i])
				continue
			}
			if acks != nil {
				if seq, ok := extractAckSeq(frame); ok {
					acks.observe(seq)
				}
			}
			frame = frame[:0]
			acks.flush(context.TODO(), conn)
		}
		if err != nil {
			if acks.active() && isTimeout(err) {
				acks.flush(context.TODO(), conn)
				continue
			}
			return
		}
	}
}

func ackTestService(t *testing.T, enabled bool) *Service {
	t.Helper()
	s := new(Service)
	s.Cfg.Log.AckProtocolEnabled = enabled
	return s
}

// A message dropped by the oversized guard is discarded as it streams: its sequence number
// cannot be read. It still reached the CDN, so it is acknowledged like a rejected line is,
// otherwise the hole stalls the contiguous ack for the rest of the connection and the client
// replays the message on every reconnection, only to see it dropped again each time.
func TestAckState_ObserveDroppedFillsTheHole(t *testing.T) {
	a := newAckState()

	// Not attributed before the connection proved it numbers its messages: an old client must
	// not start receiving acks because one of its messages was oversized.
	a.observeDropped()
	require.False(t, a.active())
	require.EqualValues(t, 0, a.contiguous)

	a.observe(1)
	a.observe(2)
	a.observeDropped() // the oversized message carried seq 3
	a.observe(4)
	require.EqualValues(t, 4, a.contiguous)

	// Nil state (protocol disabled): a no-op, not a panic.
	var disabled *ackState
	disabled.observeDropped()
}

// The budget exists so that an idle ack wake-up (read deadline expired, nothing read) does not
// burn the process-global tcp rate limit: thousands of quiet opted-in connections would starve
// the real log traffic, and the stalled reads would trip the clients' dead pipe timers into a
// replay storm. A read that returns data pays exactly once.
func TestReadBudget_IdleWakeUpsDoNotDrainTheLimiter(t *testing.T) {
	var payments int
	budget := &readBudget{wait: func(n int) error { payments++; return nil }}

	// First iteration pays.
	require.NoError(t, budget.ensure(1024))
	require.Equal(t, 1, payments)

	// Ten idle wake-ups: the deadline expired, nothing was read, the credit is kept.
	for i := 0; i < 10; i++ {
		budget.consumed(0)
		require.NoError(t, budget.ensure(1024))
	}
	require.Equal(t, 1, payments)

	// A read that returns data consumes the credit: the next iteration pays again.
	budget.consumed(512)
	require.NoError(t, budget.ensure(1024))
	require.Equal(t, 2, payments)
}

// A failed payment must not be treated as a credit, or a limiter error would let the next read
// through for free.
func TestReadBudget_FailedPaymentIsNotACredit(t *testing.T) {
	var payments int
	failing := true
	budget := &readBudget{wait: func(n int) error {
		payments++
		if failing {
			return errors.New("rate limiter closed")
		}
		return nil
	}}

	require.Error(t, budget.ensure(1024))
	failing = false
	require.NoError(t, budget.ensure(1024))
	require.Equal(t, 2, payments)
}

// The two sides of the protocol, over a real socket: the writer of the worker against the ack
// loop of the CDN. Neither unit test can catch a disagreement between the two implementations,
// this one can.
func TestAckProtocol_EndToEnd(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go serveAcks(ackTestService(t, true), ln)

	w, err := graylog.NewTCPWriter(ln.Addr().String(), nil)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		require.NoError(t, w.WriteMessage(&graylog.Message{
			Version: "1.1", Host: "worker", Short: "hello", Level: 6,
		}))
	}

	require.Eventually(t, func() bool { return w.Stats().Acked == 3 }, 5*time.Second, 10*time.Millisecond,
		"the worker must have released its replay ring from the acks written by the CDN")
}

// Flag off is the default and must be a platform wide no-op: a worker that numbers its messages
// keeps sending them and never receives anything back, so it never arms and behaves exactly as
// it did before the protocol existed.
func TestAckProtocol_DisabledByFlag(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go serveAcks(ackTestService(t, false), ln)

	w, err := graylog.NewTCPWriter(ln.Addr().String(), nil)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		require.NoError(t, w.WriteMessage(&graylog.Message{
			Version: "1.1", Host: "worker", Short: "hello", Level: 6,
		}))
	}

	// Long enough for any ack to have been written, and for a read deadline to have fired
	time.Sleep(300 * time.Millisecond)
	require.Equal(t, uint64(0), w.Stats().Acked, "no ack may be written when the protocol is disabled")

	// And the connection is still perfectly usable: the worker keeps streaming
	require.NoError(t, w.WriteMessage(&graylog.Message{
		Version: "1.1", Host: "worker", Short: "still alive", Level: 6,
	}))
	require.Equal(t, uint64(4), w.Stats().Sent)
}

// The flag is off unless the configuration says otherwise: a missing key in the TOML reads as
// false, which is exactly what is wanted for an experimental protocol.
func TestNewConnectionAckState_OffByDefault(t *testing.T) {
	var s Service
	require.Nil(t, s.newConnectionAckState(), "no configuration at all must mean no ack")
	require.False(t, s.newConnectionAckState().active(), "and the read path must tolerate that nil state")

	s.Cfg.Log.AckProtocolEnabled = true
	require.NotNil(t, s.newConnectionAckState())
}

// Protocol enabled, but a worker that does not number its messages: it must get exactly the
// previous behavior, nothing written back and no deadline.
func TestAckState_StaysInertForAnOldWorker(t *testing.T) {
	a := newAckState()
	require.False(t, a.enabled)
	require.False(t, a.due())

	server, client := net.Pipe()
	t.Cleanup(func() { _ = server.Close(); _ = client.Close() })
	a.flush(context.TODO(), server) // must not write anything, nor block

	_ = client.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	buf := make([]byte, 8)
	_, err := client.Read(buf)
	require.Error(t, err, "nothing must have been written to an old worker")
}
