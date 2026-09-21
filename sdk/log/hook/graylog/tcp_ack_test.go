package graylog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeServer is the CDN side of a net.Pipe: it reads NUL terminated frames and lets a test
// write acks back. net.Pipe is synchronous, so it must keep reading for the writer to progress.
type fakeServer struct {
	conn net.Conn

	mu     sync.Mutex
	frames [][]byte
	closed bool
}

func newFakeServer(t *testing.T, conn net.Conn) *fakeServer {
	s := &fakeServer{conn: conn}
	go s.read()
	t.Cleanup(s.close)
	return s
}

func (s *fakeServer) read() {
	var buf []byte
	chunk := make([]byte, 256)
	for {
		n, err := s.conn.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
			for {
				i := bytes.IndexByte(buf, 0)
				if i < 0 {
					break
				}
				frame := make([]byte, i)
				copy(frame, buf[:i])
				s.mu.Lock()
				s.frames = append(s.frames, frame)
				s.mu.Unlock()
				buf = buf[i+1:]
			}
		}
		if err != nil {
			return
		}
	}
}

func (s *fakeServer) ack(t *testing.T, seq uint64) {
	_, err := s.conn.Write([]byte(fmt.Sprintf(`{"ack":%d}`+"\x00", seq)))
	require.NoError(t, err)
}

func (s *fakeServer) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		_ = s.conn.Close()
	}
}

// received returns the sequence numbers and short messages of everything the server got.
func (s *fakeServer) received(t *testing.T) []receivedMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make([]receivedMessage, 0, len(s.frames))
	for _, f := range s.frames {
		var m struct {
			Short string `json:"short_message"`
			Seq   uint64 `json:"_ack_seq"`
		}
		require.NoError(t, json.Unmarshal(f, &m), "server received an invalid frame: %s", string(f))
		res = append(res, receivedMessage{Seq: m.Seq, Short: m.Short})
	}
	return res
}

func (s *fakeServer) waitFor(t *testing.T, count int) []receivedMessage {
	t.Helper()
	for i := 0; i < 200; i++ {
		if msgs := s.received(t); len(msgs) >= count {
			return msgs
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.FailNowf(t, "timeout", "expected %d messages, got %d", count, len(s.received(t)))
	return nil
}

type receivedMessage struct {
	Seq   uint64
	Short string
}

// newTestWriter returns a writer whose connections are net.Pipes handed over by the test, with
// timings short enough for a unit test.
func newTestWriter(t *testing.T, conns ...net.Conn) *TCPWriter {
	w := new(TCPWriter)
	w.Hostname = "test"
	w.maxPendingMessages = defaultMaxPendingMessages
	w.maxPendingBytes = defaultMaxPendingBytes
	w.ackReadTick = 10 * time.Millisecond
	w.ackDeadPipeTimeout = 100 * time.Millisecond

	var mu sync.Mutex
	var i int
	w.connect = func() (net.Conn, error) {
		mu.Lock()
		defer mu.Unlock()
		if i >= len(conns) {
			return nil, fmt.Errorf("no more connection available")
		}
		c := conns[i]
		i++
		return c, nil
	}
	return w
}

func msg(short string) *Message {
	return &Message{Version: "1.1", Host: "test", Short: short, Level: 6}
}

// Every message of a connection carries a 1 based sequence number: without it the CDN has
// nothing to acknowledge and the worker nothing to release.
func TestTCPWriter_SequenceNumbersArePerConnection(t *testing.T) {
	client, server := net.Pipe()
	srv := newFakeServer(t, server)
	w := newTestWriter(t, client)

	for _, s := range []string{"one", "two", "three"} {
		require.NoError(t, w.WriteMessage(msg(s)))
	}

	require.Equal(t, []receivedMessage{
		{Seq: 1, Short: "one"},
		{Seq: 2, Short: "two"},
		{Seq: 3, Short: "three"},
	}, srv.waitFor(t, 3))
	require.Equal(t, 3, w.pendingCount(), "nothing is acknowledged yet, everything stays replayable")
}

// An ack releases every message up to its sequence number: that is what keeps the replay ring
// from growing, and it is cumulative so a lost ack costs nothing.
func TestTCPWriter_AckReleasesPendingMessages(t *testing.T) {
	client, server := net.Pipe()
	srv := newFakeServer(t, server)
	w := newTestWriter(t, client)

	for _, s := range []string{"one", "two", "three"} {
		require.NoError(t, w.WriteMessage(msg(s)))
	}
	srv.waitFor(t, 3)

	srv.ack(t, 2)
	require.Eventually(t, func() bool { return w.pendingCount() == 1 }, time.Second, 5*time.Millisecond)
	require.Equal(t, uint64(2), w.Stats().Acked)
}

// The reason the ring exists: what the CDN never acknowledged is written again on the next
// connection, in order, before anything new.
func TestTCPWriter_ReplaysUnackedMessagesOnReconnect(t *testing.T) {
	client1, server1 := net.Pipe()
	client2, server2 := net.Pipe()
	srv1 := newFakeServer(t, server1)
	srv2 := newFakeServer(t, server2)
	w := newTestWriter(t, client1, client2)

	for _, s := range []string{"one", "two", "three"} {
		require.NoError(t, w.WriteMessage(msg(s)))
	}
	srv1.waitFor(t, 3)

	// Only the first message made it to the storage as far as the worker knows
	srv1.ack(t, 1)
	require.Eventually(t, func() bool { return w.pendingCount() == 2 }, time.Second, 5*time.Millisecond)

	// The connection breaks: the write of "four" fails on the broken pipe, but it was already
	// tracked, so it is replayed too instead of being lost.
	srv1.close()
	require.Error(t, w.WriteMessage(msg("four")))
	require.NoError(t, w.WriteMessage(msg("five")))

	require.Equal(t, []receivedMessage{
		{Seq: 1, Short: "two"},
		{Seq: 2, Short: "three"},
		{Seq: 3, Short: "four"},
		{Seq: 4, Short: "five"},
	}, srv2.waitFor(t, 4), "the unacked messages are replayed first, renumbered on the new connection")
	require.Equal(t, uint64(3), w.Stats().Replayed)
}

// A worker must never trade its memory for log durability: past the bound the oldest unacked
// messages are evicted, and the eviction is counted.
func TestTCPWriter_RingBoundEvictsOldest(t *testing.T) {
	client, server := net.Pipe()
	srv := newFakeServer(t, server)
	w := newTestWriter(t, client)
	w.maxPendingMessages = 2

	for _, s := range []string{"one", "two", "three", "four"} {
		require.NoError(t, w.WriteMessage(msg(s)))
	}
	srv.waitFor(t, 4)

	require.Equal(t, 2, w.pendingCount())
	require.Equal(t, uint64(2), w.Stats().Evicted)
	require.Equal(t, uint64(0), w.Stats().EvictedUnconfirmed,
		"no ack was ever received: the messages may well have been stored, we just cannot confirm it")
}

// A CDN that does not know the protocol says nothing at all. The worker must keep sending, not
// declare the pipe dead: dead pipe detection is only armed once a first ack has been seen.
func TestTCPWriter_DeadPipeOnlyArmedAfterFirstAck(t *testing.T) {
	client, server := net.Pipe()
	srv := newFakeServer(t, server)
	w := newTestWriter(t, client)

	require.NoError(t, w.WriteMessage(msg("one")))
	srv.waitFor(t, 1)

	// Well past the dead pipe timeout, with a message still unacked
	time.Sleep(4 * w.ackDeadPipeTimeout)
	require.NoError(t, w.WriteMessage(msg("two")), "an old CDN must not look like a broken one")

	got := srv.waitFor(t, 2)
	require.Equal(t, uint64(2), got[1].Seq, "same connection, the sequence simply goes on")
	require.Equal(t, 2, w.pendingCount(), "nothing was acked, so nothing was released")
}

// Once the CDN has acked at least once, silence means the pipe is dead: the connection is
// closed so that the next write reconnects and replays.
func TestTCPWriter_DeadPipeClosesConnectionOnceArmed(t *testing.T) {
	client1, server1 := net.Pipe()
	client2, server2 := net.Pipe()
	srv1 := newFakeServer(t, server1)
	srv2 := newFakeServer(t, server2)
	w := newTestWriter(t, client1, client2)

	require.NoError(t, w.WriteMessage(msg("one")))
	srv1.waitFor(t, 1)
	srv1.ack(t, 1) // arms the detection

	require.NoError(t, w.WriteMessage(msg("two")))
	srv1.waitFor(t, 2)

	// "two" is never acked: the reader must give up on this connection
	require.Eventually(t, func() bool {
		_, err := client1.Write([]byte{0})
		return err != nil
	}, 2*time.Second, 10*time.Millisecond, "the reader goroutine must have closed the dead connection")

	// And the next write reconnects and replays what was not acknowledged
	_ = w.WriteMessage(msg("three"))
	require.NoError(t, w.WriteMessage(msg("four")))
	got := srv2.waitFor(t, 2)
	require.Equal(t, "two", got[0].Short, "the unacked message is replayed on the new connection")
	require.Equal(t, uint64(1), got[0].Seq)
}

func TestWithAckSeq(t *testing.T) {
	require.Equal(t, `{"a":1,"_ack_seq":7}`+"\x00", string(withAckSeq([]byte(`{"a":1}`), 7)))
	require.Equal(t, `{"_ack_seq":1}`+"\x00", string(withAckSeq([]byte(`{}`), 1)))
	require.Equal(t, "not json\x00", string(withAckSeq([]byte("not json"), 3)))
}

func TestParseAck(t *testing.T) {
	seq, ok := parseAck([]byte(`{"ack":42}`))
	require.True(t, ok)
	require.Equal(t, uint64(42), seq)

	_, ok = parseAck([]byte(`{"ack":0}`))
	require.False(t, ok)

	_, ok = parseAck([]byte(`garbage`))
	require.False(t, ok, "anything that is not an ack is ignored, the server may write what it wants")
}
