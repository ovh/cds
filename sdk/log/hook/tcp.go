// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
// inspired from github.com/gemnasium/logrus-graylog-hook

package hook

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/eapache/go-resiliency/retrier"
)

// TCPWriter implements io.Writer and is used to send both discret
// messages to a graylog2 server, or data from a stream-oriented
// interface (like the functions in log).
type TCPWriter struct {
	mu      sync.Mutex
	conn    net.Conn
	connect func() (net.Conn, error)

	Hostname string
	Facility string
}

// NewTCPWriter returns a new TCP GELF Writer.  This writer can be used to send the
// output of the standard Go log functions to a central GELF server by
// passing it to log.SetOutput()
func NewTCPWriter(addr string, tlsCfg *tls.Config) (*TCPWriter, error) {
	var err error
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

	if err != nil {
		return nil, err
	}

	// Get Hostname if possible, otherwise just set to localhost
	if w.Hostname, err = os.Hostname(); err != nil {
		w.Hostname = "localhost"
	}

	// Set facility to binary name
	w.Facility = path.Base(os.Args[0])

	return w, nil
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
		Time:     float64(time.Now().UnixNano()) / 1E9,
		Level:    6, // info
		Facility: w.Facility,
		File:     file,
		Line:     line,
		Extra:    map[string]interface{}{},
	}

	if err := w.WriteMessage(&m); err != nil {
		return 0, err
	}

	return len(p), nil
}

// WriteMessage writes a GELF message with current TCP connection
func (w *TCPWriter) WriteMessage(m *Message) error {
	mBytes, err := json.Marshal(m)
	if err != nil {
		// should never fail
		return err
	}
	mBytes = append(mBytes, byte(0))

	w.mu.Lock()
	defer w.mu.Unlock()
	conn, err := w.get()
	if err != nil {
		return err
	}

	var n, nn int
	for n, err = conn.Write(mBytes); n < len(mBytes) && err == nil; {
		nn, err = conn.Write(mBytes[n:])
		n += nn
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "[gelf] error while sending message:", err)
		if cerr := w.conn.Close(); cerr != nil {
			fmt.Fprintln(os.Stderr, "[gelf] connection close error:", cerr)
		}
		w.conn = nil
		return err
	}

	return nil
}

// get returns a connection, reconnecting if needed.
// w.connect MUST be set and w.mu must be held
func (w *TCPWriter) get() (c net.Conn, err error) {
	if w.conn != nil {
		c = w.conn
	}

	// Try 30 times, with 1 second interval, to connect to graylog endpoint.
	// This could take up to a minute to execute (30 * (1 second delay + 1 second dial timeout)).
	err = retrier.New(retrier.ConstantBackoff(30, time.Second), nil).Run(func() error {
		if c == nil {
			fmt.Fprintln(os.Stderr, "[gelf] connecting to logging server")
			conn, err := w.connect()
			if err != nil {
				fmt.Fprintln(os.Stderr, "[gelf] cannot connect to logging server:", err)
				return err
			}
			fmt.Fprintln(os.Stderr, "[gelf] connection to logging server opened")
			c = conn
		}
		return nil
	})

	if err != nil {
		return
	}

	w.conn = c
	return
}
