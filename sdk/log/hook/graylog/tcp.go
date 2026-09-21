// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
// inspired from github.com/gemnasium/logrus-graylog-hook

package graylog

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eapache/go-resiliency/retrier"
)

// Application level acknowledgement of the log lines.
//
// TLS and TCP may terminate at intermediate proxies or load balancers in front of the CDN, so
// no TCP level signal ever proves the CDN received anything: a write can succeed against a
// proxy while the CDN never sees the line. The only signal that crosses the whole chain is an
// ack written back by the CDN itself on the same connection.
//
// The protocol is opt-in on both ends. The writer numbers every message of a connection with
// AckSeqField (1 based), and a CDN that understands it writes back {"ack":<n>}\0 on the same
// connection, n being the highest contiguous sequence it received. A CDN that does not
// understand it writes nothing at all, which is why dead pipe detection is only armed once a
// first ack has been seen on the connection.
const (
	// AckSeqField is the GELF extra field carrying the per connection sequence number.
	AckSeqField = "_ack_seq"

	// defaultMaxPendingMessages and defaultMaxPendingBytes bound the replay ring. A worker must
	// never trade its memory for log durability: past those bounds the oldest unacked messages
	// are evicted.
	defaultMaxPendingMessages = 4096
	defaultMaxPendingBytes    = 4 << 20 // 4MB

	// defaultAckReadTick is how often the reader goroutine stops waiting for an ack to check
	// whether the pipe went dead.
	defaultAckReadTick = time.Second

	// defaultAckDeadPipeTimeout is how long a connection may stay silent, with messages sent
	// and acks armed, before being considered dead.
	defaultAckDeadPipeTimeout = 10 * time.Second

	// maxAckFrameSize bounds what is accepted from the server between two NUL separators.
	maxAckFrameSize = 4096
)

// pendingMessage is a message that has been written but not acknowledged yet. The body is the
// marshalled GELF message WITHOUT its sequence number: a replay only has to number it again.
type pendingMessage struct {
	seq  uint64
	body []byte
}

// TCPWriterStats reports what the writer did with the messages it was given.
//
// PRODUCTION HARDENING: those counters live here because this branch is built on master, where
// the graylog hook has none. They are meant to feed the hook level accounting (emitted /
// dropped / write_errors) once both branches meet.
type TCPWriterStats struct {
	Sent     uint64 // messages written on a connection, replays excluded
	Replayed uint64 // messages written again after a reconnection
	Acked    uint64 // messages released by an ack
	Evicted  uint64 // messages evicted from the replay ring before being acked
	// EvictedUnconfirmed counts the evictions that happened while acks were armed. Those are
	// the ones that may really be lost: against a CDN that does not ack, an eviction only means
	// the delivery could not be confirmed, not that it failed.
	EvictedUnconfirmed uint64
}

// TCPWriter implements io.Writer and is used to send both discret
// messages to a graylog2 server, or data from a stream-oriented
// interface (like the functions in log).
type TCPWriter struct {
	mu      sync.Mutex
	conn    net.Conn
	connect func() (net.Conn, error)

	Hostname string
	Facility string

	// ackMu guards the acknowledgement state below. It is only ever held for O(1) updates and
	// never across an I/O: the reader goroutine must not wait for the write path, which holds
	// mu for as long as get() needs to reconnect (up to ~180s). Lock order is mu then ackMu;
	// the reader goroutine never takes mu.
	ackMu        sync.Mutex
	generation   uint64 // incremented for each new connection
	seq          uint64 // last sequence number written on the current connection
	pending      []pendingMessage
	pendingBytes int
	acksArmed    bool      // an ack has been received on the current connection
	lastAck      time.Time // when the last ack was received, or the connection opened

	// Tunables, taken from the defaults above. Fields rather than constants so that the tests
	// do not have to wait seconds, and so that they can be configured later.
	maxPendingMessages int
	maxPendingBytes    int
	ackReadTick        time.Duration
	ackDeadPipeTimeout time.Duration

	sent               atomic.Uint64
	replayed           atomic.Uint64
	acked              atomic.Uint64
	evicted            atomic.Uint64
	evictedUnconfirmed atomic.Uint64
}

// NewTCPWriter returns a new TCP GELF Writer.  This writer can be used to send the
// output of the standard Go log functions to a central GELF server by
// passing it to log.SetOutput()
func NewTCPWriter(addr string, tlsCfg *tls.Config) (*TCPWriter, error) {
	w := new(TCPWriter)

	const dialTimeout = 5 * time.Second

	// If TLS configuration is specified, try to connect with it
	if tlsCfg != nil {
		w.connect = func() (net.Conn, error) {
			return tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", addr, tlsCfg)
		}
	} else {
		w.connect = func() (net.Conn, error) { return net.DialTimeout("tcp", addr, dialTimeout) }
	}

	// Get Hostname if possible, otherwise just set to localhost
	var err error
	if w.Hostname, err = os.Hostname(); err != nil {
		w.Hostname = "localhost"
	}

	// Set facility to binary name
	w.Facility = path.Base(os.Args[0])

	w.maxPendingMessages = defaultMaxPendingMessages
	w.maxPendingBytes = defaultMaxPendingBytes
	w.ackReadTick = defaultAckReadTick
	w.ackDeadPipeTimeout = defaultAckDeadPipeTimeout

	return w, nil
}

// Stats returns the accounting of the writer.
func (w *TCPWriter) Stats() TCPWriterStats {
	return TCPWriterStats{
		Sent:               w.sent.Load(),
		Replayed:           w.replayed.Load(),
		Acked:              w.acked.Load(),
		Evicted:            w.evicted.Load(),
		EvictedUnconfirmed: w.evictedUnconfirmed.Load(),
	}
}

// Write writes a given data, converts it to a GELF message and writes it with
// the current TCP connection
func (w *TCPWriter) Write(p []byte) (int, error) {
	// 1 for the function that called us.
	file, line := getCallerIgnoringLogMulti(1)

	// remove trailing and leading whitespace
	p = bytes.TrimSpace(p)

	// If there are newlines in the message, use the first line
	// for the short message and set the full message to the
	// original input.  If the input has no newlines, stick the
	// whole thing in Short.
	short := p
	full := []byte("")
	if i := bytes.IndexRune(p, '\n'); i > 0 {
		short = p[:i]
		full = p
	}

	m := Message{
		Version:  "1.1",
		Host:     w.Hostname,
		Short:    string(short),
		Full:     string(full),
		Time:     float64(time.Now().UnixNano()) / 1e9,
		Level:    6, // info
		Facility: w.Facility,
		File:     file,
		Line:     line,
		Extra:    map[string]interface{}{},
	}

	if err := w.WriteMessage(&m); err != nil {
		fmt.Fprintln(os.Stderr, "[gelf] Try 1 retry: ", err)
		if err := w.WriteMessage(&m); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

// WriteMessage writes a GELF message with current TCP connection
func (w *TCPWriter) WriteMessage(m *Message) error {
	body, err := json.Marshal(m)
	if err != nil {
		// should never fail
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	conn, isNew, err := w.get()
	if err != nil {
		return err
	}
	if isNew {
		// Sequence numbers are per connection: the new one starts again at 1 and everything
		// that was not acknowledged on the previous one is written again first, in order.
		w.startConnection(conn)
		if err := w.replay(conn); err != nil {
			return err
		}
	}

	seq := w.track(body)
	if err := w.writeFrame(conn, withAckSeq(body, seq)); err != nil {
		return err
	}
	w.sent.Add(1)
	return nil
}

// writeFrame writes one NUL terminated frame. w.mu must be held.
func (w *TCPWriter) writeFrame(conn net.Conn, frame []byte) error {
	var n, nn int
	var err error
	for n, err = conn.Write(frame); n < len(frame) && err == nil; {
		nn, err = conn.Write(frame[n:])
		n += nn
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "[gelf] error while sending message:", err)
		if w.conn != nil {
			if cerr := w.conn.Close(); cerr != nil {
				fmt.Fprintln(os.Stderr, "[gelf] connection close error:", cerr)
			}
		}
		w.conn = nil
		return err
	}

	return nil
}

// track numbers the message on the current connection and keeps it for a possible replay. The
// oldest messages are evicted when the ring is full. w.mu must be held.
func (w *TCPWriter) track(body []byte) uint64 {
	w.ackMu.Lock()
	defer w.ackMu.Unlock()

	w.seq++
	w.pending = append(w.pending, pendingMessage{seq: w.seq, body: body})
	w.pendingBytes += len(body)

	// A message larger than the whole ring is written but not kept: the loop stops on an empty
	// ring instead of spinning.
	for len(w.pending) > 0 && (len(w.pending) > w.maxPendingMessages || w.pendingBytes > w.maxPendingBytes) {
		w.pendingBytes -= len(w.pending[0].body)
		w.pending = w.pending[1:]
		w.evicted.Add(1)
		if w.acksArmed {
			w.evictedUnconfirmed.Add(1)
		}
	}

	return w.seq
}

// replay writes again, in order and with fresh sequence numbers, every message that was not
// acknowledged on the previous connection. w.mu must be held, and it must be called right
// after startConnection: no ack can have been received yet on this connection.
func (w *TCPWriter) replay(conn net.Conn) error {
	w.ackMu.Lock()
	replayed := make([]pendingMessage, len(w.pending))
	for i := range w.pending {
		w.seq++
		w.pending[i].seq = w.seq
		replayed[i] = w.pending[i]
	}
	w.ackMu.Unlock()

	for i := range replayed {
		if err := w.writeFrame(conn, withAckSeq(replayed[i].body, replayed[i].seq)); err != nil {
			return err
		}
		w.replayed.Add(1)
	}
	return nil
}

// startConnection resets the per connection state and starts the goroutine reading the acks.
// The replay ring is kept: that is the whole point. w.mu must be held.
func (w *TCPWriter) startConnection(conn net.Conn) {
	w.ackMu.Lock()
	w.generation++
	generation := w.generation
	w.seq = 0
	w.acksArmed = false
	w.lastAck = time.Now()
	w.ackMu.Unlock()

	go w.readAcks(conn, generation)
}

// readAcks consumes the acks sent back by the CDN. It only reads from conn: on a dead pipe or
// a read error it closes the connection and returns, which makes the next write fail and go
// through the existing reconnect path. It never takes w.mu.
func (w *TCPWriter) readAcks(conn net.Conn, generation uint64) {
	var buf []byte
	chunk := make([]byte, 256)

	for {
		if w.currentGeneration() != generation {
			return
		}

		// The deadline is what makes this loop check for a dead pipe even when the server says
		// nothing at all, which is exactly what an old CDN does.
		_ = conn.SetReadDeadline(time.Now().Add(w.ackReadTick))
		n, err := conn.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
			for {
				i := bytes.IndexByte(buf, 0)
				if i < 0 {
					break
				}
				if seq, ok := parseAck(buf[:i]); ok {
					w.release(generation, seq)
				}
				buf = buf[i+1:]
			}
			// Garbage without any separator: resynchronize instead of growing forever.
			if len(buf) > maxAckFrameSize {
				buf = buf[:0]
			}
		}
		if err == nil {
			continue
		}

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			if !w.deadPipe(generation) {
				continue
			}
			fmt.Fprintf(os.Stderr, "[gelf] no ack received for %s, closing the connection\n", w.ackDeadPipeTimeout)
		}

		// Either the pipe is dead, or the connection is already gone. Closing here is what
		// triggers the reconnection: the write path owns w.conn, not this goroutine.
		_ = conn.Close()
		return
	}
}

// release drops every message acknowledged by the server and arms the dead pipe detection. An
// ack of a previous connection is ignored: sequence numbers restart at 1 on each connection.
func (w *TCPWriter) release(generation, ack uint64) {
	w.ackMu.Lock()
	defer w.ackMu.Unlock()

	if generation != w.generation {
		return
	}

	w.acksArmed = true
	w.lastAck = time.Now()

	var released int
	for released < len(w.pending) && w.pending[released].seq <= ack {
		w.pendingBytes -= len(w.pending[released].body)
		released++
	}
	if released > 0 {
		w.pending = w.pending[released:]
		w.acked.Add(uint64(released))
	}
}

// deadPipe reports whether messages are waiting for an ack that does not come. It stays false
// as long as no ack has ever been received on the connection: a CDN that does not know the
// protocol must not look like a broken one.
func (w *TCPWriter) deadPipe(generation uint64) bool {
	w.ackMu.Lock()
	defer w.ackMu.Unlock()

	if generation != w.generation || !w.acksArmed || len(w.pending) == 0 {
		return false
	}
	return time.Since(w.lastAck) > w.ackDeadPipeTimeout
}

func (w *TCPWriter) currentGeneration() uint64 {
	w.ackMu.Lock()
	defer w.ackMu.Unlock()
	return w.generation
}

// pendingCount returns the number of messages waiting for an acknowledgement.
func (w *TCPWriter) pendingCount() int {
	w.ackMu.Lock()
	defer w.ackMu.Unlock()
	return len(w.pending)
}

// get returns a connection, reconnecting if needed, and whether it just opened it.
// w.connect MUST be set and w.mu must be held
func (w *TCPWriter) get() (net.Conn, bool, error) {
	if w.conn != nil {
		return w.conn, false, nil
	}

	var c net.Conn
	// Try 30 times, with 1 second interval, to connect to graylog endpoint.
	// This could take up to a minute to execute (30 * (1 second delay + 1 second dial timeout)).
	err := retrier.New(retrier.ConstantBackoff(30, time.Second), nil).Run(func() error {
		fmt.Fprintln(os.Stderr, "[gelf] connecting to logging server")
		conn, err := w.connect()
		if err != nil {
			fmt.Fprintln(os.Stderr, "[gelf] cannot connect to logging server:", err)
			return err
		}
		fmt.Fprintln(os.Stderr, "[gelf] connection to logging server opened")
		c = conn
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	w.conn = c
	return c, true, nil
}

// withAckSeq appends the sequence number to an already marshalled GELF message and terminates
// the frame with the NUL separator used by the GELF TCP framing. The message is not marshalled
// again: a replay only changes this number.
func withAckSeq(body []byte, seq uint64) []byte {
	trimmed := bytes.TrimRight(body, " \t\r\n")
	if len(trimmed) < 2 || trimmed[len(trimmed)-1] != '}' {
		// Not a JSON object: send it untouched rather than corrupting it.
		return append(append([]byte{}, body...), 0)
	}

	frame := make([]byte, 0, len(trimmed)+32)
	frame = append(frame, trimmed[:len(trimmed)-1]...)
	if len(trimmed) > 2 {
		frame = append(frame, ',')
	}
	frame = append(frame, '"')
	frame = append(frame, AckSeqField...)
	frame = append(frame, '"', ':')
	frame = strconv.AppendUint(frame, seq, 10)
	frame = append(frame, '}', 0)
	return frame
}

type ackFrame struct {
	Ack uint64 `json:"ack"`
}

// parseAck reads one ack frame. Anything else received on the connection is ignored: the
// server side of this protocol is optional, so whatever else it may write is not our business.
func parseAck(frame []byte) (uint64, bool) {
	frame = bytes.TrimSpace(frame)
	if len(frame) == 0 {
		return 0, false
	}
	var f ackFrame
	if err := json.Unmarshal(frame, &f); err != nil {
		return 0, false
	}
	return f.Ack, f.Ack > 0
}
