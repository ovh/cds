// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"io"
	"net"
	"sync"
	"time"
)

// Conn is an in-memory implementation of net.Conn.
type Conn struct {
	mu *sync.Mutex   // Control mutex
	r  *halfConn     // Local side
	w  *halfConn     // Remote side
	rd time.Time     // Read deadline
	wd time.Time     // Write deadline
	t  time.Duration // Read/write timeout
}

// NewConn creates a pair of connected net.Conn instances. The addresses are
// arbitrary strings used to distinguish the two ends of the connection. bufSize
// is the maximum number of bytes that can be written to each connection before
// Write will block. A value <= 0 for bufSize means use a default of 4096 bytes.
func NewConn(addrA, addrB string, bufSize int) (A *Conn, B *Conn) {
	if bufSize <= 0 {
		bufSize = 4096
	}
	mu := new(sync.Mutex)
	a := newHalfConn(mu, addrA, bufSize)
	b := newHalfConn(mu, addrB, bufSize)
	return &Conn{mu: mu, r: a, w: b}, &Conn{mu: mu, r: b, w: a}
}

// Read reads data from the connection. It can be made to time out and return a
// net.Error with Timeout() == true after a deadline or a per-Read timeout; see
// SetDeadline, SetReadDeadline, and SetTimeout.
func (c *Conn) Read(b []byte) (n int, err error) {
	var t timer
	c.mu.Lock()
	defer c.mu.Unlock()
	t.Set(c.rd, c.t)
	if n, err = c.r.read(b, &t, c.r.addr); err == io.EOF {
		c.close()
	}
	return
}

// Write writes data to the connection. It can be made to time out and return a
// net.Error with Timeout() == true after a deadline or a per-Write timeout; see
// SetDeadline, SetWriteDeadline, and SetTimeout.
func (c *Conn) Write(b []byte) (n int, err error) {
	var t timer
	c.mu.Lock()
	defer c.mu.Unlock()
	t.Set(c.wd, c.t)
	return c.w.write(b, &t, c.r.addr)
}

// Close closes the connection. Any blocked Read or Write operations will be
// unblocked and return errors.
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.close()
	return nil
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return netAddr(c.r.addr)
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return netAddr(c.w.addr)
}

// SetDeadline sets the Read and Write deadlines associated with the connection.
// It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
func (c *Conn) SetDeadline(t time.Time) error {
	c.mu.Lock()
	c.rd = t
	c.wd = t
	c.mu.Unlock()
	return nil
}

// SetReadDeadline sets the deadline for future Read calls. A zero value for t
// means Read will not time out (but see SetTimeout).
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	c.rd = t
	c.mu.Unlock()
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls. Even if Write
// times out, it may return n > 0, indicating that some of the data was
// successfully written. A zero value for t means Write will not time out (but
// see SetTimeout).
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	c.wd = t
	c.mu.Unlock()
	return nil
}

// SetTimeout sets the per-call timeout for future Read and Write calls. It
// works in addition to any configured deadlines. A value <= 0 for d means
// Read and Write will not time out (unless a deadline is set).
func (c *Conn) SetTimeout(d time.Duration) error {
	if d < 0 {
		d = 0
	}
	c.mu.Lock()
	c.t = d
	c.mu.Unlock()
	return nil
}

// close closes the connection.
func (c *Conn) close() {
	if c.r.buf != nil {
		c.r.buf = nil
		c.r.eof = true
		c.r.Broadcast()

		c.w.eof = true
		c.w.Broadcast()

		c.rd = time.Time{}
		c.wd = time.Time{}
		c.t = 0
	}
}

// halfConn implements a unidirectional data pipe.
type halfConn struct {
	sync.Cond

	addr string // Reader's address
	buf  []byte // Read/write buffer
	off  int    // Read offset in buf
	eof  bool   // Writer closed flag
}

// newHalfConn creates a new halfConn instance.
func newHalfConn(mu *sync.Mutex, addr string, bufSize int) *halfConn {
	return &halfConn{
		Cond: *sync.NewCond(mu),
		addr: addr,
		buf:  make([]byte, 0, bufSize),
	}
}

// read copies data from the buffer into b.
func (c *halfConn) read(b []byte, t *timer, addr string) (n int, err error) {
	for {
		switch {
		case c.buf == nil:
			return n, io.EOF
		case t.Expired():
			return n, netTimeout("mock.Conn(" + addr + "): read timeout")
		case len(b) == 0:
			return
		case len(c.buf) > 0:
			n = copy(b, c.buf[c.off:])
			if c.off += n; c.off == len(c.buf) {
				c.buf = c.buf[:0]
				c.off = 0
				if c.eof {
					return n, io.EOF
				}
			}
			c.Broadcast()
			return
		}
		if t.Schedule(&c.Cond) {
			defer t.Stop()
		}
		c.Wait()
	}
}

// write copies data from b into the buffer.
func (c *halfConn) write(b []byte, t *timer, addr string) (n int, err error) {
	for {
		switch {
		case c.eof:
			return n, io.EOF
		case t.Expired():
			return n, netTimeout("mock.Conn(" + addr + "): write timeout")
		case len(b) == 0:
			return
		case len(c.buf) == 0:
			c.buf = c.buf[:copy(c.buf[:cap(c.buf)], b)]
			c.Broadcast()
			if n += len(c.buf); len(b) == len(c.buf) {
				return
			}
			b = b[len(c.buf):]
		default:
			if free := cap(c.buf) - len(c.buf); free < len(b) {
				if free += c.off; free < len(b) {
					break // Will block anyway, let the reader(s) catch up
				}
				c.buf = c.buf[:copy(c.buf, c.buf[c.off:])]
				c.off = 0
			}
			n += copy(c.buf[len(c.buf):cap(c.buf)], b)
			c.buf = c.buf[:len(c.buf)+len(b)]
			return
		}
		if t.Schedule(&c.Cond) {
			defer t.Stop()
		}
		c.Wait()
	}
}

// timer interrupts blocked Read/Write calls at a specific deadline or after a
// per-call timeout, whichever is earlier.
type timer struct {
	*time.Timer
	now, rem int64
}

// Set configures timer expiration parameters.
func (t *timer) Set(deadline time.Time, timeout time.Duration) {
	if dnz := !deadline.IsZero(); dnz || timeout > 0 {
		t.now = time.Now().UnixNano()
		t.rem = int64(timeout)
		if dnz {
			dt := deadline.UnixNano() - t.now
			if timeout <= 0 || dt < int64(timeout) {
				t.rem = dt
			}
		}
	}
}

// Expired returns true if the timer has expired.
func (t *timer) Expired() bool {
	if t.now != 0 {
		if t.Timer != nil {
			now := time.Now().UnixNano()
			t.rem -= now - t.now
			t.now = now
		}
		return t.rem <= 0
	}
	return false
}

// Schedule configures the timer to call c.Broadcast() when the timer expires.
// It returns true if the caller should defer a call to t.Stop().
func (t *timer) Schedule(c *sync.Cond) bool {
	if t.now != 0 {
		if t.Timer == nil {
			t.Timer = time.AfterFunc(time.Duration(t.rem), func() {
				// c.L must be held to guarantee that the caller is waiting
				c.L.Lock()
				defer c.L.Unlock()
				c.Broadcast()
			})
			return true
		}
		t.Reset(time.Duration(t.rem))
	}
	return false
}

// netAddr implements net.Addr for the "mock" network.
type netAddr string

func (netAddr) Network() string  { return "mock" }
func (a netAddr) String() string { return string(a) }

// netTimeout implements net.Error with Timeout() == true.
type netTimeout string

func (t netTimeout) Error() string { return string(t) }
func (netTimeout) Timeout() bool   { return true }
func (netTimeout) Temporary() bool { return true }
