// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"bufio"
	"compress/flate"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
)

// Labels for identifying the source of log entries.
const (
	client = 'C'
	server = 'S'
)

// Limit for the maximum number of characters to print in messages containing
// raw command/response lines.
const rawLimit = 1024

// ProtocolError indicates a low-level problem with the data being sent by the
// client or server.
type ProtocolError struct {
	Info string // Short message explaining the problem
	Line []byte // Full or partial command/response line
}

func (err *ProtocolError) Error() string {
	if err.Line == nil {
		return "imap: " + err.Info
	}
	line, ellipsis := err.Line, ""
	if len(line) > rawLimit {
		line, ellipsis = line[:rawLimit], "..."
	}
	return fmt.Sprintf("imap: %s (%+q%s)", err.Info, line, ellipsis)
}

// Errors returned by the low-level data transport.
var (
	ErrCompressionActive = errors.New("imap: compression already enabled")
	ErrEncryptionActive  = errors.New("imap: encryption already enabled")
)

// BufferSize sets the size of the send and receive buffers (in bytes). This is
// also the length limit of physical lines. In practice, the client should
// restrict line length to approximately 1000 bytes, as described in RFC 2683.
var BufferSize = 65536

// Line termination.
var crlf = []byte{cr, lf}

// ioLink redirects Read/Write operations when compression or encryption become
// enabled. It also keeps track of how many bytes have been transferred via the
// link.
type ioLink struct {
	io.Reader
	io.Writer

	// Read/write byte count
	rc, wc int64
}

func (l *ioLink) Attach(r io.Reader, w io.Writer) {
	l.Reader = r
	l.Writer = w
}

func (l *ioLink) Read(p []byte) (n int, err error) {
	n, err = l.Reader.Read(p)
	l.rc += int64(n)
	return
}

func (l *ioLink) Write(p []byte) (n int, err error) {
	n, err = l.Writer.Write(p)
	l.wc += int64(n)
	return
}

func (l *ioLink) Flush() error {
	type Flusher interface {
		Flush() error
	}
	if f, ok := l.Writer.(Flusher); ok {
		return f.Flush()
	}
	return nil
}

func (l *ioLink) Close() error {
	var rerr error
	if r, ok := l.Reader.(io.Closer); ok {
		rerr = r.Close()
	}
	if w, ok := l.Writer.(io.Closer); ok {
		if werr := w.Close(); werr != nil {
			return werr
		}
	}
	return rerr
}

// transport handles low-level communications with the IMAP server. It supports
// optional compression and encryption, which can be enabled at any time and in
// any order. Buffering is provided for incoming and outgoing data. The complete
// data flow is shown in the following diagram (bracketed stages are optional):
//
// 	transport <--> buffer <--> [compression] <--> [encryption] <--> network
type transport struct {
	buf     *bufio.ReadWriter // I/O buffer
	bufLink *ioLink           // Buffer Read/Write provider
	cmpLink *ioLink           // Compression Read/Write provider
	conn    net.Conn          // Network connection

	// Debug logging
	*debugLog
}

// newTransport wraps an existing network connection in a new transport
// instance. The connection may already be encrypted.
func newTransport(conn net.Conn, log *debugLog) *transport {
	lnk := &ioLink{Reader: conn, Writer: conn}
	buf := bufio.NewReadWriter(
		bufio.NewReaderSize(lnk, BufferSize),
		bufio.NewWriterSize(lnk, BufferSize),
	)
	return &transport{buf: buf, bufLink: lnk, conn: conn, debugLog: log}
}

// Compressed returns true if data compression is enabled.
func (t *transport) Compressed() bool {
	return t.cmpLink != nil
}

// Encrypted returns true if data encryption is enabled.
func (t *transport) Encrypted() bool {
	_, ok := t.conn.(*tls.Conn)
	return ok
}

// Closed returns true after Close is called on the transport.
func (t *transport) Closed() bool {
	return t.conn == nil
}

// ReadLine returns the next physical line received from the server. The CRLF
// ending is stripped and err is set to nil if and only if the line ends with
// CRLF, and does not contain NUL, CR, or LF characters anywhere else in the
// text. Otherwise, all bytes that have been read are returned unmodified along
// with an error explaining the problem.
func (t *transport) ReadLine() (line []byte, err error) {
	line, err = t.buf.ReadSlice(lf)
	n := len(line)

	// Copy bytes out of the read buffer
	if n > 0 {
		temp := make([]byte, n)
		copy(temp, line)
		line = temp
	} else {
		line = nil
	}

	// Check line format; if err == nil, the line ends with LF
	if err == nil {
		if n >= 2 && line[n-2] == cr {
			line = line[:n-2]
			for _, c := range line {
				if c < ctl && (c == nul || c == cr) {
					line = line[:n]
					err = &ProtocolError{"bad line format", line}
					break
				}
			}
		} else {
			err = &ProtocolError{"bad line ending", line}
		}
	} else if err == bufio.ErrBufferFull {
		err = &ProtocolError{"line too long", line}
	}
	t.LogLine(server, line, err)
	return
}

// WriteLine writes a physical line to the internal buffer. The CRLF ending is
// appended automatically. The line will not be sent to the server until Flush
// is called or the buffer becomes full from subsequent writes.
func (t *transport) WriteLine(line []byte) error {
	var err error

	// Check line format
	for _, c := range line {
		if c < ctl && (c == nul || c == cr || c == lf) {
			err = &ProtocolError{"bad line format", line}
			break
		}
	}

	// Free enough space in the buffer for the entire line
	if n := len(line) + 2; n > t.buf.Available() && err == nil {
		if err = t.buf.Flush(); n > t.buf.Available() && err == nil {
			err = &ProtocolError{"line too long", line}
		}
	}

	// Write the line followed by CRLF
	if err == nil {
		if _, err = t.buf.Write(line); err == nil {
			_, err = t.buf.Write(crlf)
		}
	}
	t.LogLine(client, line, err)
	return err
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read
// (0 <= n <= len(p)) and any error encountered.
func (t *transport) Read(p []byte) (n int, err error) {
	n, err = t.buf.Read(p)
	t.LogBytes(server, n, err)
	return
}

// Write writes len(p) bytes from p to the internal buffer. It returns the
// number of bytes written from p (0 <= n <= len(p)) and any error encountered
// that caused the write to stop early.
func (t *transport) Write(p []byte) (n int, err error) {
	n, err = t.buf.Write(p)
	t.LogBytes(client, n, err)
	return
}

// Flush sends any buffered data to the server.
func (t *transport) Flush() error {
	err := t.buf.Flush()
	if t.Compressed() && err == nil {
		err = t.bufLink.Flush()
	}
	return err
}

// EnableDeflate turns on DEFLATE compression. See flate.NewWriter for
// information about compression levels.
func (t *transport) EnableDeflate(level int) error {
	if t.Compressed() {
		return ErrCompressionActive
	}
	conn := &ioLink{Reader: t.conn, Writer: t.conn}
	inflater := flate.NewReader(conn)
	deflater, err := flate.NewWriter(conn, level)

	if err == nil {
		t.cmpLink = conn
		t.bufLink.Attach(inflater, deflater)
		t.Logf(LogConn, "DEFLATE compression enabled (level=%d)", level)
	}
	return err
}

// EnableTLS turns on TLS encryption.
func (t *transport) EnableTLS(config *tls.Config) error {
	if t.Encrypted() {
		return ErrEncryptionActive
	}
	conn := tls.Client(t.conn, config)
	if err := conn.Handshake(); err != nil {
		t.Logf(LogConn, "TLS handshake failed (%v)", err)
		return err
	}

	t.conn = conn
	if t.Compressed() {
		t.cmpLink.Attach(conn, conn)
	} else {
		t.bufLink.Attach(conn, conn)
	}
	state := conn.ConnectionState()
	t.Logf(LogConn, "TLS encryption enabled (cipher=0x%04X)", state.CipherSuite)
	return nil
}

// Close terminates the connection. If flush == true, any buffered data is sent
// out before the connection is closed. Calling Flush followed by Close(false)
// is not the same as calling Close(true). Only use Close(false) when no further
// communications are possible (e.g. other side already closed the connection).
func (t *transport) Close(flush bool) error {
	if t.Closed() {
		return nil
	}
	conn := t.conn
	t.conn = nil
	t.Logf(LogConn, "Connection closing (flush=%v)", flush)

	if flush {
		err := t.buf.Flush()
		if t.Compressed() && err == nil {
			err = t.bufLink.Close()
		}
		if err != nil {
			conn.Close()
			return err
		}
	}
	return conn.Close()
}

// LogLine logs a physical line transfer from the client or server.
func (t *transport) LogLine(src byte, line []byte, err error) {
	if t.debugLog == nil || t.debugLog.mask&LogRaw != LogRaw {
		return
	}
	ellipsis := ""
	if len(line) > rawLimit {
		line, ellipsis = line[:rawLimit], "..."
	}
	if err == nil {
		t.Logf(LogRaw, "%c: %s%s", src, line, ellipsis)
		return
	}
	var info string
	if pe, ok := err.(*ProtocolError); ok {
		info = pe.Info
	} else {
		info = err.Error()
	}
	t.Logf(LogRaw, "%c: %s%s (%s)", src, line, ellipsis, info)
}

// LogBytes logs a literal byte transfer from the client or server.
func (t *transport) LogBytes(src byte, n int, err error) {
	if t.debugLog == nil || t.debugLog.mask&LogRaw != LogRaw {
		return
	}
	if err == nil {
		t.Logf(LogRaw, "%c: literal %d bytes", src, n)
		return
	}
	t.Logf(LogRaw, "%c: literal %d bytes (%v)", src, n, err)
}
