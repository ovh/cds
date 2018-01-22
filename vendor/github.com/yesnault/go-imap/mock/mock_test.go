// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock_test

import (
	"io"
	"strings"
	"testing"

	"github.com/mxk/go-imap/imap"
	"github.com/mxk/go-imap/mock"
)

func init() {
	imap.DefaultLogMask = imap.LogRaw
}

func TestGreeting(T *testing.T) {
	// Typical greeting followed by the CAPABILITY command
	t := mock.Server(T,
		`S: * OK Server ready`,
		`C: A1 CAPABILITY`,
		`S: * CAPABILITY IMAP4rev1`,
		`S: A1 OK Thats all she wrote!`,
	)
	_, err := t.Dial()
	t.Join(err)

	// Capabilities sent in the greeting
	t = mock.Server(T,
		`S: * OK [CAPABILITY IMAP4rev1] Server ready`,
	)
	_, err = t.Dial()
	t.Join(err)

	// TLS negotiated before the greeting
	t = mock.Server(T,
		mock.STARTTLS,
		`S: * PREAUTH [CAPABILITY IMAP4rev1] Server ready`,
	)
	_, err = t.DialTLS(nil)
	t.Join(err)

	// Connection refused
	t = mock.Server(T,
		`S: * BYE Server not ready`,
		mock.CLOSE,
	)
	if _, err = t.Dial(); err == nil {
		t.Errorf("t.Dial() expected an error")
	}
	t.Join(nil)
}

func TestSession(T *testing.T) {
	t := mock.Server(T,
		`S: * OK [CAPABILITY IMAP4rev1 STARTTLS LOGINDISABLED] Server ready`,
	)
	c, err := t.Dial()
	t.Join(err)

	// STARTTLS
	t.Script(
		`C: A1 STARTTLS`,
		`S: A1 OK Begin TLS negotiation now`,
		mock.STARTTLS,
		`C: A2 CAPABILITY`,
		`S: * CAPABILITY IMAP4rev1`,
		`S: A2 OK Thats all she wrote!`,
	)
	t.Join(t.StartTLS(nil))

	// LOGIN
	t.Script(
		`C: A3 LOGIN "joe" "password"`,
		`S: A3 OK LOGIN completed`,
		`C: A4 CAPABILITY`,
		`S: * CAPABILITY IMAP4rev1 COMPRESS=DEFLATE`,
		`S: A4 OK Thats all she wrote!`,
	)
	_, err = c.Login("joe", "password")
	t.Join(err)

	// COMPRESS
	t.Script(
		`C: A5 COMPRESS DEFLATE`,
		`S: A5 OK DEFLATE active`,
		mock.DEFLATE,
	)
	_, err = c.CompressDeflate(-1)
	t.Join(err)

	// LOGOUT
	t.Script(
		`C: A6 LOGOUT`,
		`S: * BYE LOGOUT Requested`,
		`S: A6 OK Quoth the raven, nevermore...`,
		mock.CLOSE,
	)
	_, err = c.Logout(mock.Timeout)
	t.Join(err)

	// Verify EOF
	if err = c.Recv(mock.Timeout); err != io.EOF {
		t.Fatalf("c.Recv() expected EOF; got %v", err)
	}
}

func TestLiteral(T *testing.T) {
	t := mock.Server(T,
		`S: * PREAUTH [CAPABILITY IMAP4rev1] Server ready`,
	)
	c, err := t.Dial()
	t.Join(err)

	flags := imap.NewFlagSet(`\Seen`)
	lines := []string{
		"Date: Mon, 7 Feb 1994 21:52:25 -0800 (PST)",
		"From: Fred Foobar <foobar@Blurdybloop.COM>",
		"Subject: afternoon meeting",
		"To: mooch@owatagu.siam.edu",
		"Message-Id: <B27397-0100000@Blurdybloop.COM>",
		"MIME-Version: 1.0",
		"Content-Type: TEXT/PLAIN; CHARSET=US-ASCII",
		"",
		"Hello Joe, do you think we can meet at 3:30 tomorrow?",
		"",
	}
	msg := []byte(strings.Join(lines, "\r\n"))
	lit := imap.NewLiteral(msg)

	// Embedded literal
	t.Script(
		`C: A1 APPEND "saved-messages" (\Seen) {310}`,
		`S: + Ready for literal data`,
		`C: `+lines[0],
		`C: `+lines[1],
		`C: `+lines[2],
		`C: `+lines[3],
		`C: `+lines[4],
		`C: `+lines[5],
		`C: `+lines[6],
		`C: `+lines[7],
		`C: `+lines[8],
		`C: `+lines[9],
		`S: A1 OK APPEND completed`,
	)
	_, err = imap.Wait(c.Append("saved-messages", flags, nil, lit))
	t.Join(err)

	// Recv action literal
	t.Script(
		`C: A2 APPEND "saved-messages" (\Seen) {310}`,
		`S: + Ready for literal data`,
		mock.Recv(msg),
		`C: `,
		`S: A2 OK APPEND completed`,
	)
	_, err = imap.Wait(c.Append("saved-messages", flags, nil, lit))
	t.Join(err)

	// Embedded and Send action literals from the server
	t.Script(
		`C: A3 LIST "" "*"`,
		`S: * LIST (\Noselect) "/" {3}`,
		`S: foo`,
		`S: * LIST () "/" {7}`,
		mock.Send("foo/bar"),
		`S: `,
		`S: A3 OK LIST completed`,
	)
	_, err = imap.Wait(c.List("", "*"))
	t.Join(err)
}
