// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package mock implements a scripted IMAP server for testing client behavior.

The mock server understands low-level details of the IMAP protocol (lines,
literals, compression, encryption, etc.). It doesn't know anything about
commands, users, mailboxes, messages, or any other high-level concepts. The
server follows a script that tells it what to send/receive and when. Everything
received from the client is checked against the script and an error is returned
if there is a mismatch.

See mock_test.go for examples of how to use this package in your unit tests.
*/
package mock

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mxk/go-imap/imap"
)

// ServerName is the hostname used by the scripted server.
var ServerName = "imap.mock.net"

// Timeout is the maximum execution time for each Read and Write call on the
// simulated network connection. When the client or server aborts execution due
// to an unexpected error, this timeout prevents the other side from blocking
// indefinitely.
var Timeout = 500 * time.Millisecond

// Send and Recv are script actions for sending and receiving raw bytes (usually
// literal strings).
type (
	Send []byte
	Recv []byte
)

// ScriptFunc is function type called during script execution to control the
// server state. STARTTLS, DEFLATE, and CLOSE are predefined script actions for
// the most common operations.
type ScriptFunc func(s imap.MockServer) error

// Predefined script actions for controlling server state.
var (
	STARTTLS = func(s imap.MockServer) error { return s.EnableTLS(serverTLS()) }
	DEFLATE  = func(s imap.MockServer) error { return s.EnableDeflate(-1) }
	CLOSE    = func(s imap.MockServer) error { return s.Close(true) }
)

// T wraps existing test state and provides methods for testing the IMAP client
// against the scripted server.
type T struct {
	*testing.T

	s  imap.MockServer    // Server instance
	ch <-chan interface{} // Script result channel

	c  *imap.Client // Client instance
	cn net.Conn     // Client connection used by Dial and DialTLS
}

// Server launches a new scripted server that can handle one client connection.
// The script should contain the initial server greeting. It should also use the
// STARTTLS action (or a custom ScriptFunc for negotiating encryption) prior to
// sending the greeting if the client is using DialTLS to connect.
func Server(t *testing.T, script ...interface{}) *T {
	c, s := NewConn("client", "server", 0)
	c.SetTimeout(Timeout)
	s.SetTimeout(Timeout)
	mt := &T{T: t, s: imap.NewMockServer(s), cn: c}
	mt.Script(script...)
	return mt
}

// Dial returns a new Client connected to the scripted server or an error if the
// connection could not be established.
func (t *T) Dial() (*imap.Client, error) {
	cn := t.cn
	if t.cn = nil; cn == nil {
		t.Fatalf(cl("t.Dial() or t.DialTLS() already called for this server"))
	}
	var err error
	if t.c, err = imap.NewClient(cn, ServerName, Timeout); err != nil {
		cn.Close()
	}
	return t.c, err
}

// DialTLS returns a new Client connected to the scripted server or an error if
// the connection could not be established. The server is expected to negotiate
// encryption before sending the initial greeting. Config should be nil when
// used in combination with the predefined STARTTLS script action.
func (t *T) DialTLS(config *tls.Config) (*imap.Client, error) {
	cn := t.cn
	if t.cn = nil; cn == nil {
		t.Fatalf(cl("t.Dial() or t.DialTLS() already called for this server"))
	}
	if config == nil {
		config = clientTLS()
	}
	tlsConn := tls.Client(cn, config)
	var err error
	if t.c, err = imap.NewClient(tlsConn, ServerName, Timeout); err != nil {
		cn.Close()
	}
	return t.c, err
}

// Script runs a server script in a separate goroutine. A script is a sequence
// of string, Send, Recv, and ScriptFunc actions. Strings represent lines of
// text to be sent ("S: ...") or received ("C: ...") by the server. There is an
// implicit CRLF at the end of each line. Send and Recv allow the server to send
// and receive raw bytes (usually literal strings). ScriptFunc allows server
// state changes by calling methods on the provided imap.MockServer instance.
func (t *T) Script(script ...interface{}) {
	select {
	case <-t.ch:
	default:
		if t.ch != nil {
			t.Fatalf(cl("t.Script() called while another script is active"))
		}
	}
	ch := make(chan interface{}, 1)
	t.ch = ch
	go t.script(script, ch)
}

// Join waits for script completion and reports any errors encountered by the
// client or the server.
func (t *T) Join(err error) {
	if err, ok := <-t.ch; err != nil {
		t.Errorf(cl("t.Join() S: %v"), err)
	} else if !ok {
		t.Errorf(cl("t.Join() called without an active script"))
	}
	if err != nil {
		t.Fatalf(cl("t.Join() C: %v"), err)
	} else if t.Failed() {
		t.FailNow()
	}
}

// StartTLS performs client-side TLS negotiation. Config should be nil when used
// in combination with the predefined STARTTLS script action.
func (t *T) StartTLS(config *tls.Config) error {
	if t.c == nil {
		t.Fatalf(cl("t.StartTLS() called without a valid client"))
	}
	if config == nil {
		config = clientTLS()
	}
	_, err := t.c.StartTLS(config)
	return err
}

// script runs the provided script and sends the first encountered error to ch,
// which is then closed.
func (t *T) script(script []interface{}, ch chan<- interface{}) {
	defer func() { ch <- recover(); close(ch) }()
	for ln, v := range script {
		switch ln++; v := v.(type) {
		case string:
			if strings.HasPrefix(v, "S: ") {
				err := t.s.WriteLine([]byte(v[3:]))
				t.flush(ln, v, err)
			} else if strings.HasPrefix(v, "C: ") {
				b, err := t.s.ReadLine()
				t.compare(ln, v[3:], string(b), err)
			} else {
				panicf(`[#%d] %+q must be prefixed with "S: " or "C: "`, ln, v)
			}
		case Send:
			_, err := t.s.Write(v)
			t.flush(ln, v, err)
		case Recv:
			b := make([]byte, len(v))
			_, err := io.ReadFull(t.s, b)
			t.compare(ln, string(v), string(b), err)
		case ScriptFunc:
			t.run(ln, v)
		case func(s imap.MockServer) error:
			t.run(ln, v)
		default:
			panicf("[#%d] %T is not a valid script action", ln, v)
		}
	}
}

// flush sends any buffered data to the client and panics if there is an error.
func (t *T) flush(ln int, v interface{}, err error) {
	if err == nil {
		err = t.s.Flush()
	}
	if err != nil {
		panicf("[#%d] %+q write error: %v", ln, v, err)
	}
}

// compare panics if v != b or err != nil.
func (t *T) compare(ln int, v, b string, err error) {
	if v != b || err != nil {
		panicf("[#%d] expected %+q; got %+q (%v)", ln, v, b, err)
	}
}

// run calls v and panics if it returns an error.
func (t *T) run(ln int, v ScriptFunc) {
	if err := v(t.s); err != nil {
		panicf("[#%d] ScriptFunc error: %v", ln, err)
	}
}

// cl prefixes s with the current line number in the calling test function.
func cl(s string) string {
	_, testFile, line, ok := runtime.Caller(2)
	if ok && strings.HasSuffix(testFile, "_test.go") {
		return fmt.Sprintf("%d: %s", line, s)
	}
	return s
}

// panicf must be documented for consistency (you're welcome)!
func panicf(format string, v ...interface{}) {
	panic(fmt.Sprintf(format, v...))
}
