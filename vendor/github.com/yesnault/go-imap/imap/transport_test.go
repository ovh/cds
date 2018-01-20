// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"testing"
	"time"
)

// Timeout for testConn read/write operations.
var testConnTimeout = 100 * time.Millisecond

// testConn implements net.Conn via buffered channels.
type testConn struct {
	r <-chan byte
	w chan<- byte
}

// testConnError implements net.Error.
type testConnError struct {
	err     string
	timeout bool
	temp    bool
}

func (err *testConnError) Error() string   { return err.err }
func (err *testConnError) Timeout() bool   { return err.timeout }
func (err *testConnError) Temporary() bool { return err.temp }

// testAddr implements net.Addr.
type testAddr string

// newTestConn returns two testConn instances representing two sides of a
// network connection.
func newTestConn(bufSize int) (*testConn, *testConn) {
	R, W := make(chan byte, bufSize), make(chan byte, bufSize)
	return &testConn{r: R, w: W}, &testConn{r: W, w: R}
}
func (c *testConn) Read(b []byte) (n int, err error) {
	if ok := false; len(b) > 0 {
		select {
		case b[n], ok = <-c.r:
			if !ok {
				return 0, io.EOF
			}
			n++
		case <-time.After(testConnTimeout):
			return 0, &testConnError{err: "testConn: read timeout", timeout: true}
		}
		for ok && n < len(b) {
			select {
			case b[n], ok = <-c.r:
				if ok {
					n++
				}
			default:
				ok = false
			}
		}
	}
	return
}
func (c *testConn) Write(b []byte) (n int, err error) {
	if c.w == nil {
		return 0, &testConnError{err: "testConn: write to a closed conn"}
	} else if len(b) > 0 {
		timeout := time.After(testConnTimeout)
		for ; n < len(b); n++ {
			select {
			case c.w <- b[n]:
			case <-timeout:
				return n, &testConnError{err: "testConn: write timeout", timeout: true}
			}
		}
	}
	return
}
func (c *testConn) Clear() {
	for ok := true; ok; {
		select {
		case _, ok = <-c.r:
		default:
			ok = false
		}
	}
}
func (c *testConn) Close() error {
	if c.w != nil {
		close(c.w)
		c.w = nil
	}
	return nil
}
func (c *testConn) LocalAddr() net.Addr                { return testAddr("192.0.2.2:32687") }
func (c *testConn) RemoteAddr() net.Addr               { return testAddr("192.0.2.1:143") }
func (c *testConn) SetDeadline(t time.Time) error      { return nil }
func (c *testConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *testConn) SetWriteDeadline(t time.Time) error { return nil }

func (a testAddr) Network() string { return "test-net-1" }
func (a testAddr) String() string  { return string(a) }

// String-based wrappers for transport methods.
func (t *transport) readln() (s string, err error) {
	b, err := t.ReadLine()
	return string(b), err
}
func (t *transport) writeln(ln string) error {
	return t.WriteLine([]byte(ln))
}
func (t *transport) read(n int) (s string, err error) {
	b := make([]byte, n)
	n, err = io.ReadFull(t, b)
	return string(b[:n]), err
}
func (t *transport) write(b string) error {
	_, err := t.Write([]byte(b))
	return err
}
func (t *transport) send(v ...string) error {
	literal := false
	for _, in := range v {
		if literal {
			if err := t.write(in); err != nil {
				return err
			}
			literal = false
		} else {
			if err := t.writeln(in); err != nil {
				return err
			}
			literal = len(in) > 0 && in[len(in)-1] == '}'
		}
	}
	return t.Flush()
}
func (t *transport) clear() {
	if v, ok := t.conn.(*testConn); ok {
		v.Clear()
	}
	t.Read(make([]byte, t.buf.Reader.Buffered()))
}
func (t *transport) starttls(client bool) (err error) {
	if client {
		err = t.EnableTLS(tlsConfig.client)
	} else {
		conn := tls.Server(t.conn, tlsConfig.server)
		if err = conn.Handshake(); err == nil {
			t.conn = conn
			if t.Compressed() {
				t.cmpLink.Attach(conn, conn)
			} else {
				t.bufLink.Attach(conn, conn)
			}
		}
	}
	return
}

// TLS client and server configuration.
var tlsConfig = struct {
	client *tls.Config
	server *tls.Config
}{}

func init() {
	var err error
	if tlsConfig.client, tlsConfig.server, err = tlsNewConfig(); err != nil {
		panic(err)
	}
}
func tlsNewConfig() (client, server *tls.Config, err error) {
	now := time.Now()
	tpl := x509.Certificate{
		SerialNumber:          new(big.Int).SetInt64(0),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(5 * time.Minute).UTC(),
		BasicConstraintsValid: true,
		IsCA: true,
	}
	priv, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		return
	}
	crt, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &priv.PublicKey, priv)
	if err != nil {
		return
	}
	key := x509.MarshalPKCS1PrivateKey(priv)
	pair, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: crt}),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: key}),
	)
	if err != nil {
		return
	}
	root, err := x509.ParseCertificate(crt)
	if err == nil {
		server = &tls.Config{Certificates: []tls.Certificate{pair}}
		client = &tls.Config{RootCAs: x509.NewCertPool(), ServerName: "localhost"}
		client.RootCAs.AddCert(root)
	}
	return
}

func TestTransportConn(t *testing.T) {
	c, s := newTestConn(8)

	// Zero read/write
	if n, err := c.Write(nil); n != 0 || err != nil {
		t.Fatalf("c.Write(nil) expected 0; got %v (%v)", n, err)
	}
	if n, err := c.Read(nil); n != 0 || err != nil {
		t.Fatalf("c.Read(nil) expected 0; got %v (%v)", n, err)
	}

	// Read timeout
	if n, err := c.Read([]byte{0}); n != 0 || err == nil {
		t.Fatalf("c.Read([]byte{0}) expected 0 (timeout); got %v (%v)", n, err)
	}

	// Partial read/write
	in := "Abc"
	if n, err := s.Write([]byte(in)); n != len(in) || err != nil {
		t.Fatalf("s.Write(%q) expected %v; got %v (%v)", in, len(in), n, err)
	}
	in, b := "A", []byte{0}
	for i := 0; i < 2; i++ {
		if n, err := c.Read(b); n != len(b) || err != nil || string(b) != in {
			t.Fatalf("c.Read(b) expected %q; got %q (%v)", in, b, err)
		}
		in, b = "bc", []byte{0, 0}
	}

	// Full read/write
	in = "username"
	if n, err := s.Write([]byte(in)); n != len(in) || err != nil {
		t.Fatalf("s.Write(%q) expected %v; got %v (%v)", in, len(in), n, err)
	}
	b = make([]byte, 10)
	if n, err := c.Read(b); n != len(in) || err != nil || string(b[:n]) != in {
		t.Fatalf("c.Read(b) expected %q; got %q (%v)", in, b[:n], err)
	}

	// Write timeout
	in = "password*!"
	if n, err := s.Write([]byte(in)); n != 8 || err == nil {
		t.Fatalf("s.Write(%q) expected 8 (timeout); got %v (%v)", in, n, err)
	}
	in = "password"
	if n, err := c.Read(b); n != len(in) || err != nil || string(b[:n]) != in {
		t.Fatalf("c.Read(b) expected %q; got %q (%v)", in, b[:n], err)
	}

	// Read timeout
	if n, err := c.Read(b); n != 0 || err == nil {
		t.Fatalf("c.Read(b) expected 0 (timeout); got %v (%v)", n, err)
	}

	// Clear
	s.Write([]byte("1 2 3"))
	c.Clear()
	if n, err := c.Read(b); n != 0 || err == nil {
		t.Fatalf("c.Read(b) expected 0 (timeout); got %v (%v)", n, err)
	}

	// Close and EOF
	s.Close()
	if n, err := s.Write([]byte(in)); n != 0 || err == nil {
		t.Fatalf("s.Write(%q) expected 0 (closed); got %v (%v)", in, n, err)
	}
	if n, err := c.Read(b); n != 0 || err != io.EOF {
		t.Fatalf("c.Read(b) expected 0 (EOF); got %v (%v)", n, err)
	}
}

func TestTransportBasic(t *testing.T) {
	c, s := newTestConn(1024)
	C, S := newTransport(c, nil), newTransport(s, nil)

	// Transport status
	if C.Compressed() {
		t.Error("C.Compressed() expected false")
	}
	if C.Encrypted() {
		t.Error("C.Encrypted() expected false")
	}
	if C.Closed() {
		t.Error("C.Closed() expected false")
	}

	// Write greeting
	in := "* IMAP4rev1 Server ready"
	if err := S.writeln(in); err != nil {
		t.Fatalf("S.writeln(%q) unexpected error; %v", in, err)
	}

	// Client doesn't receive anything until Flush is called
	select {
	case <-c.r:
		t.Fatal("<-c.r should have blocked")
	default:
	}

	// Flush greeting
	if err := S.Flush(); err != nil {
		t.Fatalf("S.Flush() unexpected error; %v", err)
	}

	// Receive greeting
	if out, err := C.readln(); out != in || err != nil {
		t.Fatalf("C.readln() expected %q; got %q (%v)", in, out, err)
	}

	tCAPABILITY(t, C, S, "A001")
	tLOGIN(t, C, S, "A002")
	tLOGOUT(t, C, S, "A003")
}

func TestTransportDeflate(t *testing.T) {
	c, s := newTestConn(1024)
	C, S := newTransport(c, nil), newTransport(s, nil)

	tGREETING(t, C, S)
	tCAPABILITY(t, C, S, "B001")
	tDEFLATE(t, C, S, "B002")
	tLOGIN(t, C, S, "B003")
	tLOGOUT(t, C, S, "B004")
}

func TestTransportTLS(t *testing.T) {
	c, s := newTestConn(1024)
	C, S := newTransport(c, nil), newTransport(s, nil)

	tGREETING(t, C, S)
	tCAPABILITY(t, C, S, "C001")
	tSTARTTLS(t, C, S, "C002")
	tLOGIN(t, C, S, "C003")
	tLOGOUT(t, C, S, "C004")
}

func TestTransportDeflateTLS(t *testing.T) {
	c, s := newTestConn(1024)
	C, S := newTransport(c, nil), newTransport(s, nil)

	tGREETING(t, C, S)
	tCAPABILITY(t, C, S, "D001")
	tDEFLATE(t, C, S, "D002")
	tSTARTTLS(t, C, S, "D003")
	tLOGIN(t, C, S, "D004")
	tLOGOUT(t, C, S, "D005")
}

func TestTransportTLSDeflate(t *testing.T) {
	c, s := newTestConn(1024)
	C, S := newTransport(c, nil), newTransport(s, nil)

	tGREETING(t, C, S)
	tSTARTTLS(t, C, S, "E001")
	tCAPABILITY(t, C, S, "E002")
	tDEFLATE(t, C, S, "E003")
	tLOGIN(t, C, S, "E004")
	tLOGOUT(t, C, S, "E005")
}

func TestTransportErrors(t *testing.T) {
	c, s := newTestConn(1024)

	// Client will use a 16-byte buffer
	orig := BufferSize
	BufferSize = 16
	C := newTransport(c, nil)
	BufferSize = orig
	S := newTransport(s, nil)

	// Line too long (write)
	in := "hello, world!!!" // 15 + 2
	if C.send(in) == nil {
		t.Fatalf("C.send(%q) expected error", in)
	}
	if out, err := S.readln(); out != "" || err == nil {
		t.Fatalf("S.readln() expected timeout; got %q (%v)", out, err)
	}

	// Line too long (before implicit flush)
	in = "hello, world"
	for i := 0; i < 2; i++ {
		if err := C.writeln(in); err != nil {
			t.Fatalf("C.writeln(%q) unexpected error; %v", in, err)
		}
		in = "!!!" // second writeln flushes the first
	}
	in = "hello, world"
	for i := 0; i < 2; i++ {
		if out, err := S.readln(); out != in || err != nil {
			t.Fatalf("S.readln() expected %q; got %q (%v)", in, out, err)
		}
		in = "!!!"
		C.Flush()
	}

	// Line too long (read)
	in = "hello, world!!!"
	if err := S.send(in); err != nil {
		t.Fatalf("S.send(%q) unexpected error; %v", in, err)
	}
	in += "\r"
	for i := 0; i < 2; i++ {
		if out, err := C.readln(); out != in || err == nil {
			t.Fatalf("C.readln() expected %q; got %q (%v)", in, out, err)
		}
		in = "\n"
	}

	// Bad input
	tests := []string{
		"* hello\r",
		"* hello\n",
		"* hello\x00",
		"* hello\r\n",
	}
	for _, in := range tests {
		if S.writeln(in) == nil {
			t.Fatalf("S.writeln(%q) expected error", in)
		}
	}

	// Bad output
	tests = []string{
		"* hello\n",
		"* hello\r\r\n",
		"* hello\x00\n",
		"* hello\x00\r\n",
	}
	for _, in := range tests {
		S.write(in) // use write to bypass line format checks in writeln
	}
	S.Flush()
	for _, in := range tests {
		if out, err := C.readln(); out != in || err == nil {
			t.Fatalf("C.readln() expected %q (bad line); got %q (%v)", in, out, err)
		}
	}
}

func tGREETING(t *testing.T, C, S *transport) {
	// Send greeting
	in := "* IMAP4rev1 Server ready"
	if err := S.send(in); err != nil {
		t.Fatalf("S.send(%q) unexpected error; %v", in, err)
	}

	// Receive greeting
	if out, err := C.readln(); out != in || err != nil {
		t.Fatalf("C.readln() expected %q; got %q (%v)", in, out, err)
	}
}

func tCAPABILITY(t *testing.T, C, S *transport, tag string) {
	// Send command
	in := tag + " CAPABILITY"
	if err := C.send(in); err != nil {
		t.Fatalf("C.send(%q) unexpected error; %v", in, err)
	}

	// Receive command
	if out, err := S.readln(); out != in || err != nil {
		t.Fatalf("S.readln() expected %q; got %q (%v)", in, out, err)
	}

	// Execute command
	rsp := []string{
		"* CAPABILITY IMAP4rev1 STARTTLS COMPRESS=DEFLATE LITERAL+",
		tag + " OK CAPABILITY completed",
	}
	if err := S.send(rsp...); err != nil {
		t.Fatalf("S.send(rsp...) unexpected error; %v", err)
	}

	// Receive response
	for _, in := range rsp {
		if out, err := C.readln(); out != in || err != nil {
			t.Fatalf("C.readln() expected %q; got %q (%v)", in, out, err)
		}
	}
}

func tDEFLATE(t *testing.T, C, S *transport, tag string) {
	// Send command
	in := tag + " COMPRESS DEFLATE"
	if err := C.send(in); err != nil {
		t.Fatalf("C.send(%q) unexpected error; %v", in, err)
	}

	// Receive command
	if out, err := S.readln(); out != in || err != nil {
		t.Fatalf("S.readln() expected %q; got %q (%v)", in, out, err)
	}

	// Execute command
	in = tag + " OK DEFLATE active"
	if err := S.send(in); err != nil {
		t.Fatalf("S.send(%q) unexpected error; %v", in, err)
	}
	if err := S.EnableDeflate(0); err != nil {
		t.Fatalf("S.EnableDeflate(0) unexpected error; %v", err)
	}

	// Receive response
	if out, err := C.readln(); out != in || err != nil {
		t.Fatalf("C.readln() expected %q; got %q (%v)", in, out, err)
	}
	if err := C.EnableDeflate(9); err != nil {
		t.Fatalf("C.EnableDeflate(9) unexpected error; %v", err)
	}

	// Check status
	if !C.Compressed() {
		t.Error("C.Compressed() expected true")
	}
	if !S.Compressed() {
		t.Error("S.Compressed() expected true")
	}

	// Compression already enabled
	if err := C.EnableDeflate(6); err == nil {
		t.Fatal("C.EnableDeflate(6) expected error")
	}
}

func tSTARTTLS(t *testing.T, C, S *transport, tag string) {
	// Send command
	in := tag + " STARTTLS"
	if err := C.send(in); err != nil {
		t.Fatalf("C.send(%q) unexpected error; %v", in, err)
	}

	// Receive command
	if out, err := S.readln(); out != in || err != nil {
		t.Fatalf("S.readln() expected %q; got %q (%v)", in, out, err)
	}

	// Execute command
	in = tag + " OK Begin TLS negotiation now"
	if err := S.send(in); err != nil {
		t.Fatalf("S.send(%q) unexpected error; %v", in, err)
	}

	// Receive response
	if out, err := C.readln(); out != in || err != nil {
		t.Fatalf("C.readln() expected %q; got %q (%v)", in, out, err)
	}

	// Perform TLS negotiation
	result := make(chan error, 1)
	go func() {
		defer close(result)
		result <- S.starttls(false)
	}()
	if err := C.starttls(true); err != nil {
		t.Fatalf("C.EnableTLS() unexpected error; %v", err)
	}
	if err, ok := <-result; err != nil || ok != true {
		t.Fatalf("tls.Server.Handshake() unexpected error; %v", err)
	}

	// Check status
	if !C.Encrypted() {
		t.Error("C.Encrypted() expected true")
	}
	if !S.Encrypted() {
		t.Error("S.Encrypted() expected true")
	}

	// Encryption already enabled
	if err := C.EnableTLS(nil); err == nil {
		t.Fatal("C.EnableTLS(nil) expected error")
	}
}

func tLOGIN(t *testing.T, C, S *transport, tag string) {
	// Send command
	cmd := []string{
		tag + " LOGIN {11+}",
		"FRED FOOBAR",
		" {7+}",
		"fat man",
		"",
	}
	if err := C.send(cmd...); err != nil {
		t.Fatalf("C.send(cmd...) unexpected error; %v", err)
	}

	// Receive command
	literal := false
	for _, in := range cmd {
		if literal {
			if out, err := S.read(len(in)); out != in || err != nil {
				t.Fatalf("S.read(%v) expected %q; got %q (%v)", len(in), in, out, err)
			}
			literal = false
		} else {
			if out, err := S.readln(); out != in || err != nil {
				t.Fatalf("S.readln() expected %q; got %q (%v)", in, out, err)
			}
			literal = len(in) > 0 && in[len(in)-1] == '}'
		}
	}

	// Execute command
	in := tag + " OK LOGIN completed"
	if err := S.send(in); err != nil {
		t.Fatalf("S.send(%q) unexpected error; %v", in, err)
	}

	// Receive response
	if out, err := C.readln(); out != in || err != nil {
		t.Fatalf("C.readln() expected %q; got %q (%v)", in, out, err)
	}
}

func tLOGOUT(t *testing.T, C, S *transport, tag string) {
	// Send command
	in := tag + " LOGOUT"
	if err := C.send(in); err != nil {
		t.Fatalf("C.send(%q) unexpected error; %v", in, err)
	}

	// Receive command
	if out, err := S.readln(); out != in || err != nil {
		t.Fatalf("S.readln() expected %q; got %q (%v)", in, out, err)
	}

	// Execute command
	rsp := []string{
		"* BYE IMAP4rev1 Server logging out",
		tag + " OK LOGOUT completed",
	}
	for _, in := range rsp {
		if err := S.writeln(in); err != nil {
			t.Fatalf("S.writeln(%q) unexpected error; %v", in, err)
		}
	}
	if err := S.Close(true); err != nil {
		t.Fatalf("S.Close(true) unexpected error; %v", err)
	}

	// Receive response
	for _, in := range rsp {
		if out, err := C.readln(); out != in || err != nil {
			t.Fatalf("C.readln() expected %q; got %q (%v)", in, out, err)
		}
	}

	// Receive EOF and close
	if out, err := C.readln(); out != "" || err != io.EOF {
		t.Fatalf("C.readln() expected EOF; got %q (%v)", out, err)
	}
	if err := C.Close(false); err != nil {
		t.Fatalf("C.Close(false) unexpected error; %v", err)
	}

	// Check status
	if !C.Closed() {
		t.Error("C.Closed() expected true")
	}
	if !S.Closed() {
		t.Error("S.Closed() expected true")
	}
}
