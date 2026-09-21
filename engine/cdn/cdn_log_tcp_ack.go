package cdn

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/rockbears/log"
)

// Application level acknowledgement of the log lines, CDN side.
//
// A worker that knows the protocol numbers every message of its connection with ackSeqField.
// The first time such a message is seen, this connection starts writing acks back on the same
// socket: {"ack":<n>}\0, n being the highest CONTIGUOUS sequence number received. A worker that
// does not number its messages gets exactly today's behavior: nothing is ever written back.
const (
	ackSeqField = "_ack_seq"

	// defaultAckEveryMessages and defaultAckEveryDuration are the ack cadence: whichever comes
	// first. One ack per message would double the syscalls of the intake for nothing, acks are
	// cumulative.
	defaultAckEveryMessages = 100
	defaultAckEveryDuration = 2 * time.Second

	// defaultAckWriteTimeout bounds the write of an ack. A client that never reads what we send
	// must not be able to block the goroutine that reads its logs.
	defaultAckWriteTimeout = 2 * time.Second

	// maxOutOfOrderSeq bounds the memory spent tracking holes in the sequence.
	maxOutOfOrderSeq = 1024

	// maxSeqDigits bounds the sequence number parsing.
	maxSeqDigits = 19
)

// ackState is the acknowledgement state of ONE tcp connection. It is only ever used from the
// goroutine that reads that connection, so it needs no lock, and the acks are written from that
// same goroutine: no concurrent write on the socket.
type ackState struct {
	enabled bool

	// contiguous is the highest sequence number such that everything below has been received.
	// The alternative, acknowledging the highest sequence SEEN, would tell the worker to
	// release messages that fell in a hole: with the contiguous one, a hole simply stops the
	// acks from progressing and the worker replays from there on the next connection.
	contiguous uint64
	outOfOrder map[uint64]struct{}

	pending      bool // something new to acknowledge
	sinceLastAck int
	lastAckAt    time.Time

	writeFailures int
	failureLogged bool

	everyMessages int
	everyDuration time.Duration
	writeTimeout  time.Duration
}

func newAckState() *ackState {
	return &ackState{
		outOfOrder:    make(map[uint64]struct{}),
		lastAckAt:     time.Now(),
		everyMessages: defaultAckEveryMessages,
		everyDuration: defaultAckEveryDuration,
		writeTimeout:  defaultAckWriteTimeout,
	}
}

// newConnectionAckState returns the ack state of a new log connection, or nil when the protocol
// is disabled by the configuration. A nil state costs nothing on the read path: no allocation,
// no scan of the received lines, no deadline, no write. This is the single place where the
// feature flag is read.
func (s *Service) newConnectionAckState() *ackState {
	if !s.Cfg.Log.AckProtocolEnabled {
		return nil
	}
	return newAckState()
}

// active reports whether this connection is acknowledging. False on a nil state, which is how
// the disabled protocol is expressed on the read path.
func (a *ackState) active() bool {
	return a != nil && a.enabled
}

// observe records a received sequence number. It is called for every line read from the
// connection, INCLUDING the ones the intake rejects: an ack says "this line reached the CDN",
// not "this line was stored". A worker cannot fix a bad signature by sending the line again, so
// acknowledging a rejected line is what stops it from being replayed forever.
func (a *ackState) observe(seq uint64) {
	a.enabled = true
	a.pending = true
	a.sinceLastAck++

	switch {
	case seq <= a.contiguous:
		// already acknowledged, a replay of a message we had received
		return
	case seq == a.contiguous+1:
		a.contiguous = seq
		for {
			next := a.contiguous + 1
			if _, ok := a.outOfOrder[next]; !ok {
				break
			}
			delete(a.outOfOrder, next)
			a.contiguous = next
		}
	default:
		// A hole: TCP does not reorder, so this means the connection restarted its numbering
		// or something was dropped upstream. Keep it aside, it may be filled.
		if len(a.outOfOrder) < maxOutOfOrderSeq {
			a.outOfOrder[seq] = struct{}{}
		}
		// PRODUCTION HARDENING: past that bound the holes are simply forgotten, which only
		// costs a replay of what is above them on the next connection.
	}
}

// due reports whether an ack must be written now.
func (a *ackState) due() bool {
	if !a.active() || !a.pending || a.contiguous == 0 {
		return false
	}
	return a.sinceLastAck >= a.everyMessages || time.Since(a.lastAckAt) >= a.everyDuration
}

// flush writes the ack if one is due. A write that cannot complete within the deadline is
// skipped, not retried: acks are cumulative, the next one carries the same information. The
// read side is deliberately left alone, a client that does not read its acks is a client whose
// logs we still want.
func (a *ackState) flush(ctx context.Context, conn net.Conn) {
	if !a.due() {
		return
	}

	frame := make([]byte, 0, 24)
	frame = append(frame, `{"ack":`...)
	frame = strconv.AppendUint(frame, a.contiguous, 10)
	frame = append(frame, '}', 0)

	_ = conn.SetWriteDeadline(time.Now().Add(a.writeTimeout))
	_, err := conn.Write(frame)
	_ = conn.SetWriteDeadline(time.Time{})

	if err != nil {
		a.writeFailures++
		if !a.failureLogged {
			a.failureLogged = true
			log.Warn(ctx, "cdn:ack: unable to write ack to %v: %v (further failures on this connection are not logged)", conn.RemoteAddr(), err)
		}
		return
	}

	a.pending = false
	a.sinceLastAck = 0
	a.lastAckAt = time.Now()
}

// extractAckSeq looks for the sequence number in a raw gelf frame. It is a scan and not a
// second json.Unmarshal: the intake already pays for one full parse per line, and this must run
// for every line, including the ones that will be rejected.
func extractAckSeq(frame []byte) (uint64, bool) {
	needle := []byte(`"` + ackSeqField + `":`)
	i := bytes.Index(frame, needle)
	if i < 0 {
		return 0, false
	}

	j := i + len(needle)
	for j < len(frame) && (frame[j] == ' ' || frame[j] == '\t') {
		j++
	}

	start := j
	var seq uint64
	for j < len(frame) && frame[j] >= '0' && frame[j] <= '9' {
		if j-start >= maxSeqDigits {
			return 0, false
		}
		seq = seq*10 + uint64(frame[j]-'0')
		j++
	}
	if j == start || seq == 0 {
		return 0, false
	}
	return seq, true
}

// isTimeout reports whether the error is the read deadline firing, which is a normal event on
// an ack enabled connection: it is what lets the time based cadence run while the client is
// silent.
func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
