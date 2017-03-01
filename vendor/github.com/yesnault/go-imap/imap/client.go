// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"errors"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

// Timeout arguments for Client.recv.
const (
	block = time.Duration(-1) // Block until a complete response is received
	poll  = time.Duration(0)  // Check for buffered responses without blocking
)

// ErrTimeout is returned when an operation does not finish successfully in the
// allocated time.
var ErrTimeout = errors.New("imap: operation timeout")

// ErrExclusive is returned when an attempt is made to execute multiple commands
// in parallel, but one of the commands requires exclusive client access.
var ErrExclusive = errors.New("imap: exclusive client access violation")

// ErrNotAllowed is returned when a command cannot be issued in the current
// connection state. Client.CommandConfig[<name>].States determines valid states
// for each command.
var ErrNotAllowed = errors.New("imap: command not allowed in the current state")

// NotAvailableError is returned when the requested command, feature, or
// capability is not supported by the client and/or server. The error may be
// temporary. For example, servers should disable the LOGIN command by
// advertising LOGINDISABLED capability while the connection is unencrypted.
// Enabling encryption via STARTTLS should allow the use of LOGIN.
type NotAvailableError string

func (err NotAvailableError) Error() string {
	return "imap: not available (" + string(err) + ")"
}

// response transports the output of Client.next through the rch channel.
type response struct {
	rsp *Response
	err error
}

// Client manages a single connection to an IMAP server.
type Client struct {
	// FIFO queue for unilateral server data. The first response is the server
	// greeting. Subsequent responses are those that were rejected by all active
	// command filters. Commands documented as expecting "no specific responses"
	// (e.g. NOOP) use nil filters by default, which reject all responses.
	Data []*Response

	// Set of current server capabilities. It is updated automatically anytime
	// new capabilities are received, which could be in a data response or a
	// status response code.
	Caps map[string]bool

	// Status of the selected mailbox. It is set to nil unless the Client is in
	// the Selected state. The fields are updated automatically as the server
	// sends solicited and unsolicited status updates.
	Mailbox *MailboxStatus

	// Execution parameters of known commands. Client.Send will return an error
	// if an attempt is made to execute a command whose name does not appear in
	// this map. The server may not support all commands known to the client.
	CommandConfig map[string]*CommandConfig

	// Server host name for authentication and STARTTLS commands.
	host string

	// Current connection state. Initially set to unknown.
	state ConnState

	// Command tag generator.
	tag tagGen

	// FIFO queue of tags for the commands in progress (keys of cmds). Response
	// filtering is performed according to the command issue order to support
	// server-side ambiguity resolution, as described in RFC 3501 section 5.5.
	tags []string

	// Map of tags to Command objects. A command is "in progress" and may
	// receive responses as long as it has an entry in this map.
	cmds map[string]*Command

	// Control and response channels for the receiver goroutine. A new response
	// channel rch is created for each time-limited receive request, and is sent
	// via cch to the receiver. The receiver sends back the output of c.next via
	// rch. There can be at most one active receive request (rch != nil).
	cch chan<- chan<- *response
	rch <-chan *response

	// Low-level transport for sending commands and receiving responses.
	t *transport
	r *reader

	// Protection against multiple close calls.
	closer sync.Once

	// Debug message logging.
	*debugLog
}

// NewClient returns a new Client instance connected to an IMAP server via conn.
// The function waits for the server to send a greeting message, and then
// requests server capabilities if they weren't included in the greeting. An
// error is returned if either operation fails or does not complete before the
// timeout, which must be positive to have any effect. If an error is returned,
// it is the caller's responsibility to close the connection.
func NewClient(conn net.Conn, host string, timeout time.Duration) (c *Client, err error) {
	log := newDebugLog(DefaultLogger, DefaultLogMask)
	cch := make(chan chan<- *response, 1)

	c = &Client{
		Caps:          make(map[string]bool),
		CommandConfig: defaultCommands(),
		host:          host,
		state:         unknown,
		tag:           *newTagGen(0),
		cmds:          make(map[string]*Command),
		t:             newTransport(conn, log),
		debugLog:      log,
	}
	c.r = newReader(c.t, MemoryReader{}, string(c.tag.id))
	c.Logf(LogConn, "Connected to %v (Tag=%s)", conn.RemoteAddr(), c.tag.id)

	if err = c.greeting(timeout); err != nil {
		c.Logln(LogConn, "Greeting error:", err)
		return nil, err
	}
	c.cch = cch
	go c.receiver(cch)
	return
}

// State returns the current connection state (Login, Auth, Selected, Logout, or
// Closed). See RFC 3501 page 15 for a state diagram. The caller must continue
// receiving responses until this method returns Closed (same as c.Recv
// returning io.EOF). Failure to do so may result in memory leaks.
func (c *Client) State() ConnState {
	return c.state
}

// Send issues a new command, returning as soon as the last line is flushed from
// the send buffer. This may involve waiting for continuation requests if
// non-synchronizing literals (RFC 2088) are not supported by the server.
//
// This is the raw command interface that does not encode or perform any
// validation of the supplied fields. It should only be used for implementing
// new commands that do not change the connection state. For commands already
// supported by this package, use the provided wrapper methods instead.
func (c *Client) Send(name string, fields ...Field) (cmd *Command, err error) {
	if cmd = newCommand(c, name); cmd == nil {
		return nil, NotAvailableError(name)
	} else if cmd.config.States&c.state == 0 {
		return nil, ErrNotAllowed
	} else if len(c.tags) > 0 {
		other := c.cmds[c.tags[0]]
		if cmd.config.Exclusive || other.config.Exclusive {
			return nil, ErrExclusive
		}
	}

	// Build command
	raw, err := cmd.build(c.tag.Next(), fields)
	if err != nil {
		return nil, err
	}

	// Write first line and update command state
	c.Logln(LogCmd, ">>>", cmd)
	if err = c.t.WriteLine(raw.ReadLine()); err != nil {
		return nil, err
	}
	c.tags = append(c.tags, cmd.tag)
	c.cmds[cmd.tag] = cmd

	// Write remaining parts, flushing the transport buffer as needed
	var rsp *Response
	for i := 0; i < len(raw.literals) && err == nil; i++ {
		if rsp, err = c.checkContinue(cmd, !raw.nonsync); err == nil {
			if rsp == nil || rsp.Type == Continue {
				if _, err = raw.literals[i].WriteTo(c.t); err == nil {
					err = c.t.WriteLine(raw.ReadLine())
				}
			} else {
				err = ResponseError{rsp, "unexpected command completion"}
			}
		}
	}

	// Flush buffer after the last line
	if err == nil {
		if err = c.t.Flush(); err == nil {
			return
		}
	}
	c.done(cmd, abort)
	return nil, err
}

// Recv receives at most one response from the server, updates the client state,
// and delivers the response to its final destination (c.Data or one of the
// commands in progress). io.EOF is returned once all responses have been
// received and the connection is closed.
//
// If the timeout is negative, Recv blocks indefinitely until a response is
// received or an error is encountered. If the timeout is zero, Recv polls for
// buffered responses, returning ErrTimeout immediately if none are available.
// Otherwise, Recv blocks until a response is received or the timeout expires.
func (c *Client) Recv(timeout time.Duration) error {
	rsp, err := c.recv(timeout)
	if err == nil && !c.deliver(rsp) {
		if rsp.Type == Continue {
			err = ResponseError{rsp, "unexpected continuation request"}
		} else {
			err = ResponseError{rsp, "undeliverable response"}
		}
	}
	return err
}

// SetLiteralReader installs a custom LiteralReader implementation into the
// response receiver pipeline. It returns the previously installed LiteralReader
// instance.
func (c *Client) SetLiteralReader(lr LiteralReader) LiteralReader {
	prev := c.r.LiteralReader
	if lr != nil {
		c.r.LiteralReader = lr
	}
	return prev
}

// Quote attempts to represent v, which must be string, []byte, or fmt.Stringer,
// as a quoted string for use with Client.Send. A literal string representation
// is used if v cannot be quoted.
func (c *Client) Quote(v interface{}) Field {
	var b []byte
	var cp bool
	switch s := v.(type) {
	case string:
		b = []byte(s)
	case []byte:
		b, cp = s, true
	case fmt.Stringer:
		b = []byte(s.String())
	default:
		return nil
	}
	if q := QuoteBytes(b, false); q != nil {
		return string(q)
	} else if cp {
		b = append([]byte(nil), b...)
	}
	return NewLiteral(b)
}

// next returns the next server response obtained directly from the reader.
func (c *Client) next() (rsp *Response, err error) {
	raw, err := c.r.Next()
	if err == nil {
		rsp, err = raw.Parse()
	}
	return
}

// greeting receives the server greeting, sets initial connection state, and
// requests server capabilities if they weren't included in the greeting.
func (c *Client) greeting(timeout time.Duration) (err error) {
	if timeout > 0 {
		// If c.recv fails, c.t.conn may be nil by the time the deferred
		// function executes; keep a reference to avoid a panic.
		conn := c.t.conn
		conn.SetDeadline(time.Now().Add(timeout))
		defer func() {
			conn.SetDeadline(time.Time{})
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				err = ErrTimeout
			}
		}()
	}

	// Wait for server greeting
	rsp, err := c.recv(block)
	if err != nil {
		return
	} else if rsp.Type != Status || !c.deliver(rsp) {
		return ResponseError{rsp, "invalid server greeting"}
	}

	// Set initial connection state
	switch rsp.Status {
	case OK:
		c.setState(Login)
	case PREAUTH:
		c.setState(Auth)
	case BYE:
		c.setState(Logout)
		fallthrough
	default:
		return ResponseError{rsp, "invalid greeting status"}
	}
	c.Logln(LogConn, "Server greeting:", rsp.Info)

	// Request capabilities if not included in the greeting
	if len(c.Caps) == 0 {
		_, err = c.Capability()
	}
	return
}

// receiver runs in a separate goroutine, reading a single server response for
// each request sent on the cch channel.
func (c *Client) receiver(cch <-chan chan<- *response) {
	recv := func() (r *response) {
		defer func() {
			if err := recover(); err != nil {
				r = &response{nil, fmt.Errorf("imap: receiver panic: %v", err)}
				c.Logf(LogGo, "Receiver panic (Tag=%s): %v\n%s", c.tag.id, err, debug.Stack())
			}
		}()
		rsp, err := c.next()
		return &response{rsp, err}
	}

	c.Logf(LogGo, "Receiver started (Tag=%s)", c.tag.id)
	defer c.Logf(LogGo, "Receiver finished (Tag=%s)", c.tag.id)

	for rch := range cch {
		rch <- recv()
	}
}

// recv returns the next server response, updating the client state beforehand.
func (c *Client) recv(timeout time.Duration) (rsp *Response, err error) {
	if c.state == Closed {
		return nil, io.EOF
	} else if c.rch == nil && (timeout < 0 || c.cch == nil) {
		rsp, err = c.next()
	} else {
		if c.rch == nil {
			rch := make(chan *response, 1)
			c.cch <- rch
			c.rch = rch
		}
		var r *response
		if timeout < 0 {
			r = <-c.rch
		} else {
			select {
			case r = <-c.rch:
			default:
				if timeout == 0 {
					return nil, ErrTimeout
				}
				select {
				case r = <-c.rch:
				case <-time.After(timeout):
					return nil, ErrTimeout
				}
			}
		}
		c.rch = nil
		rsp, err = r.rsp, r.err
	}
	if err == nil {
		c.update(rsp)
	} else if rsp == nil {
		defer c.setState(Closed)
		if err != io.EOF {
			c.close("protocol error")
		} else if err = c.close("end of stream"); err == nil {
			err = io.EOF
		}
	}
	return
}

// update examines server responses and updates client state as needed.
func (c *Client) update(rsp *Response) {
	if rsp.Label == "CAPABILITY" {
		c.setCaps(rsp.Fields[1:])
		return
	}
	switch rsp.Type {
	case Data:
		if c.Mailbox == nil {
			return
		}
		switch rsp.Label {
		case "FLAGS":
			c.Mailbox.Flags.Replace(rsp.Fields[1])
		case "EXISTS":
			c.Mailbox.Messages = rsp.Value()
		case "RECENT":
			c.Mailbox.Recent = rsp.Value()
		case "EXPUNGE":
			c.Mailbox.Messages--
			if c.Mailbox.Recent > c.Mailbox.Messages {
				c.Mailbox.Recent = c.Mailbox.Messages
			}
			if c.Mailbox.Unseen == rsp.Value() {
				c.Mailbox.Unseen = 0
			}
		}
	case Status:
		switch rsp.Status {
		case BAD:
			// RFC 3501 is a bit vague on how the client is expected to react to
			// an untagged BAD response. It's probably best to close this
			// connection and open a new one; leave this up to the caller. For
			// now, abort all active commands to avoid waiting for completion
			// responses that may never come.
			c.Logln(LogCmd, "ABORT!", rsp.Info)
			c.deliver(abort)
		case BYE:
			c.Logln(LogConn, "Logout reason:", rsp.Info)
			c.setState(Logout)
		}
		fallthrough
	case Done:
		if rsp.Label == "ALERT" {
			c.Logln(LogConn, "ALERT!", rsp.Info)
			return
		} else if c.Mailbox == nil {
			return
		}
		switch selected := (c.state == Selected); rsp.Label {
		case "PERMANENTFLAGS":
			c.Mailbox.PermFlags.Replace(rsp.Fields[1])
		case "READ-ONLY":
			if selected && !c.Mailbox.ReadOnly {
				c.Logln(LogState, "Mailbox access change: RW -> RO")
			}
			c.Mailbox.ReadOnly = true
		case "READ-WRITE":
			if selected && c.Mailbox.ReadOnly {
				c.Logln(LogState, "Mailbox access change: RO -> RW")
			}
			c.Mailbox.ReadOnly = false
		case "UIDNEXT":
			c.Mailbox.UIDNext = rsp.Value()
		case "UIDVALIDITY":
			v := rsp.Value()
			if u := c.Mailbox.UIDValidity; selected && u != v {
				c.Logf(LogState, "Mailbox UIDVALIDITY change: %d -> %d", u, v)
			}
			c.Mailbox.UIDValidity = v
		case "UNSEEN":
			c.Mailbox.Unseen = rsp.Value()
		case "UIDNOTSTICKY":
			c.Mailbox.UIDNotSticky = true
		}
	}
}

// deliver saves the response to its final destination. It returns false for
// continuation requests and unknown command completions. The abort response is
// delivered to all commands in progress.
func (c *Client) deliver(rsp *Response) bool {
	if rsp.Type&(Data|Status) != 0 {
		for _, tag := range c.tags {
			cmd := c.cmds[tag]
			if filter := cmd.config.Filter; filter != nil && filter(cmd, rsp) {
				cmd.Data = append(cmd.Data, rsp)
				return true
			}
		}
		c.Data = append(c.Data, rsp)
		return true
	} else if rsp.Type == Done {
		if cmd := c.cmds[rsp.Tag]; cmd != nil {
			c.done(cmd, rsp)
			return true
		}
		c.Logln(LogCmd, "<<<", rsp.Tag, "(Unknown)")
	} else if rsp == abort {
		for _, tag := range c.tags {
			c.done(c.cmds[tag], abort)
		}
		return true
	}
	return false
}

// done completes command execution by setting cmd.result to rsp and updating
// the client's command state.
func (c *Client) done(cmd *Command, rsp *Response) {
	if cmd.result != nil {
		return
	}
	cmd.result = rsp
	if tag := cmd.tag; c.cmds[tag] != nil {
		delete(c.cmds, tag)
		if c.tags[0] == tag {
			c.tags = c.tags[1:]
		} else if n := len(c.tags); c.tags[n-1] == tag {
			c.tags = c.tags[:n-1]
		} else {
			for i, v := range c.tags {
				if v == tag {
					c.tags = append(c.tags[:i], c.tags[i+1:]...)
					break
				}
			}
		}
	}
	if rsp == abort {
		c.Logln(LogCmd, "<<<", cmd.tag, "(Abort)")
	} else {
		c.Logln(LogCmd, "<<<", rsp)
	}
}

// checkContinue returns the next continuation request or completion result of
// cmd. In synchronous mode (sync == true), it flushes the buffer and blocks
// until a continuation request or cmd completion response is received. In
// asynchronous mode, it polls for cmd completion, returning as soon as all
// buffered responses are processed. A continuation request is not expected in
// asynchronous mode and results in an error.
func (c *Client) checkContinue(cmd *Command, sync bool) (rsp *Response, err error) {
	mode := poll
	if sync {
		if err = c.t.Flush(); err != nil {
			return
		}
		mode = block
	}
	for cmd.InProgress() {
		if rsp, err = c.recv(mode); err != nil {
			if err == ErrTimeout {
				err = nil
			}
			return
		} else if !c.deliver(rsp) {
			if rsp.Type == Continue {
				if !sync {
					err = ResponseError{rsp, "unexpected continuation request"}
				}
			} else {
				err = ResponseError{rsp, "undeliverable response"}
			}
			return
		}
	}
	return cmd.Result(0)
}

// setState changes connection state and performs the associated client updates.
// If the new state is Selected, it is assumed that c.Mailbox is already set.
func (c *Client) setState(s ConnState) {
	prev := c.state
	if prev == s || prev == Closed {
		return
	}
	c.state = s
	if s != Selected {
		c.Logf(LogState, "State change: %v -> %v", prev, s)
		c.Mailbox = nil
		if s == Closed {
			if c.cch != nil {
				close(c.cch)
			}
			c.setCaps(nil)
			c.deliver(abort)
		}
	} else if c.debugLog.mask&LogState != 0 {
		mb, rw := c.Mailbox.Name, "[RW]"
		if c.Mailbox.ReadOnly {
			rw = "[RO]"
		}
		c.Logf(LogState, "State change: %v -> %v (%+q %s)", prev, s, mb, rw)
	}
}

// setCaps updates the server capability set.
func (c *Client) setCaps(caps []Field) {
	for v := range c.Caps {
		delete(c.Caps, v)
	}
	for _, f := range caps {
		if v := toUpper(AsAtom(f)); v != "" {
			c.Caps[v] = true
		} else {
			c.Logln(LogState, "Invalid capability:", f)
		}
	}
	if c.debugLog.mask&LogState != 0 {
		caps := strings.Join(c.getCaps(""), " ")
		if caps == "" {
			caps = "(none)"
		}
		c.Logln(LogState, "Capabilities:", caps)
	}
}

// getCaps returns a sorted list of capabilities that share a common prefix. The
// prefix is stripped from the returned strings.
func (c *Client) getCaps(prefix string) []string {
	caps := make([]string, 0, len(c.Caps))
	if n := len(prefix); n == 0 {
		for v := range c.Caps {
			caps = append(caps, v)
		}
	} else {
		for v := range c.Caps {
			if strings.HasPrefix(v, prefix) {
				caps = append(caps, v[n:])
			}
		}
	}
	sort.Strings(caps)
	return caps
}

// close closes the connection without sending any additional data or updating
// client state. After the first invocation this method becomes a no-op.
func (c *Client) close(reason string) (err error) {
	c.closer.Do(func() {
		if reason != "" {
			c.Logln(LogConn, "Close reason:", reason)
		}
		if err = c.t.Close(false); err != nil {
			c.Logln(LogConn, "Close error:", err)
		}
	})
	return
}
