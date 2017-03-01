// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"crypto/tls"
	"net"
)

// MockServer is an internal type exposed for use by the mock package.
type MockServer interface {
	Compressed() bool
	Encrypted() bool
	Closed() bool
	ReadLine() (line []byte, err error)
	WriteLine(line []byte) error
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Flush() error
	EnableDeflate(level int) error
	EnableTLS(config *tls.Config) error
	Close(flush bool) error
}

// NewMockServer is an internal function exposed for use by the mock package.
func NewMockServer(conn net.Conn) MockServer {
	gotest = true
	return mockServer{newTransport(conn, nil)}
}

type mockServer struct{ *transport }

func (t mockServer) EnableTLS(config *tls.Config) (err error) {
	if t.Encrypted() {
		return ErrEncryptionActive
	}
	conn := tls.Server(t.conn, config)
	if err = conn.Handshake(); err == nil {
		t.conn = conn
		if t.Compressed() {
			t.cmpLink.Attach(conn, conn)
		} else {
			t.bufLink.Attach(conn, conn)
		}
	}
	return
}
