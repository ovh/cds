// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import "errors"

// Note:
//   Most of this code was copied, with some modifications, from net/smtp. It
//   would be better if Go provided a standard package (e.g. crypto/sasl) that
//   could be shared by SMTP, IMAP, and other packages.

// ServerInfo contains information about the IMAP server with which SASL
// authentication is about to be attempted.
type ServerInfo struct {
	Name string   // Server name
	TLS  bool     // Encryption status
	Auth []string // Supported authentication mechanisms
}

// SASL is the interface for performing challenge-response authentication.
type SASL interface {
	// Start begins SASL authentication with the server. It returns the
	// authentication mechanism name and "initial response" data (if required by
	// the selected mechanism). A non-nil error causes the client to abort the
	// authentication attempt.
	//
	// A nil ir value is different from a zero-length value. The nil value
	// indicates that the selected mechanism does not use an initial response,
	// while a zero-length value indicates an empty initial response, which must
	// be sent to the server.
	Start(s *ServerInfo) (mech string, ir []byte, err error)

	// Next continues challenge-response authentication. A non-nil error causes
	// the client to abort the authentication attempt.
	Next(challenge []byte) (response []byte, err error)
}

type externalAuth []byte

// ExternalAuth returns an implementation of the EXTERNAL authentication
// mechanism, as described in RFC 4422. Authorization identity may be left blank
// to indicate that the client is requesting to act as the identity associated
// with the authentication credentials.
func ExternalAuth(identity string) SASL {
	return externalAuth(identity)
}

func (a externalAuth) Start(s *ServerInfo) (mech string, ir []byte, err error) {
	return "EXTERNAL", a, nil
}

func (a externalAuth) Next(challenge []byte) (response []byte, err error) {
	return nil, errors.New("unexpected server challenge")
}

type plainAuth []byte

// PlainAuth returns an implementation of the PLAIN authentication mechanism, as
// described in RFC 4616. Authorization identity may be left blank to indicate
// that it is the same as the username.
func PlainAuth(username, password, identity string) SASL {
	return plainAuth(identity + "\x00" + username + "\x00" + password)
}

func (a plainAuth) Start(s *ServerInfo) (mech string, ir []byte, err error) {
	if !s.TLS {
		err = NotAvailableError("AUTH=PLAIN")
	} else {
		mech, ir = "PLAIN", a
	}
	return
}

func (a plainAuth) Next(challenge []byte) (response []byte, err error) {
	return nil, errors.New("unexpected server challenge")
}
