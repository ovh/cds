// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// Script commands for controlling server actions.
const (
	RETURN   = "RETURN"   // Stop script execution
	STARTTLS = "STARTTLS" // Perform TLS negotiation
	DEFLATE  = "DEFLATE"  // Enable DEFLATE compression
	EOF      = "EOF"      // Flush buffers and close the connection
)

func init() {
	gotest = true // util.go
}

func setLogMask(mask LogMask) func() {
	DefaultLogMask = mask
	return func() { DefaultLogMask = LogNone }
}

func un(f func()) {
	f()
}

func panicf(format string, v ...interface{}) {
	panic(fmt.Sprintf(format, v...))
}

func caller() string {
	var pc uintptr
	var ok bool
	var line int

	// Find the current line number in the parent Test* function
	for name, skip := "", 1; !strings.HasPrefix(name, "Test"); skip++ {
		if pc, _, line, ok = runtime.Caller(skip); !ok {
			return "[:?]"
		}
		name = runtime.FuncForPC(pc).Name()
		if i := strings.LastIndex(name, "."); i != -1 {
			name = name[i+1:]
		}
	}
	return fmt.Sprintf("[:%d]", line)
}

type clientT struct {
	*testing.T

	C   *Client          // Client
	S   *transport       // Server (scripted)
	sch chan interface{} // Script error channel
}

func newClient(T *testing.T, script ...string) (*Client, *clientT) {
	c, s := newTestConn(1024)
	t := &clientT{
		T:   T,
		S:   newTransport(s, nil),
		sch: make(chan interface{}, 1),
	}

	var err error
	go t.script(script...)
	t.C, err = NewClient(c, "localhost", testConnTimeout) // testConn ignores the actual timeout value

	if t.C != nil {
		t.C.CommandConfig["XYZZY"] = &CommandConfig{States: Login}
	}

	// Special handling for timeout and BYE greeting
	if len(script) > 0 {
		if script[0] == RETURN {
			if t.C != nil || err != ErrTimeout {
				t.Fatalf("%s NewClient() expected timeout; got %#v (%v)", caller(), t.C, err)
			}
		} else if strings.HasPrefix(script[0], "S: * BYE") {
			if t.C != nil || err == nil {
				t.Fatalf("%s NewClient() expected error; got %#v (%v)", caller(), t.C, err)
			}
		}
		err = nil
	}
	t.join("NewClient", err)
	return t.C, t
}
func (t *clientT) script(script ...string) {
	defer func() { t.sch <- recover() }()
	var err error
	var out string
	S := t.S
	for _, in := range script {
		switch in {
		case RETURN:
			panic(nil)
		case STARTTLS:
			err = S.starttls(false)
		case DEFLATE:
			err = S.EnableDeflate(6)
		case EOF:
			err = S.Close(true)
		default:
			send := strings.HasPrefix(in, "S: ")
			text := strings.HasSuffix(in, CRLF)
			if !send && !strings.HasPrefix(in, "C: ") {
				panicf("bad script line: %+q", in)
			} else if in = in[3:]; text {
				in = in[:len(in)-2]
			}
			if send {
				if text {
					err = S.writeln(in)
				} else {
					err = S.write(in)
				}
				if err == nil {
					err = S.Flush()
				}
			} else if text {
				if out, err = S.readln(); out != in {
					panicf("S.readln() expected %+q; got %+q (%v)", in, out, err)
				}
			} else {
				n := len(in)
				if out, err = S.read(n); out != in {
					panicf("S.read(%d) expected %+q; got %+q (%v)", n, in, out, err)
				}
			}
		}
		if err != nil {
			panic(err)
		}
	}
}
func (t *clientT) join(checkpoint string, err error) {
	select {
	case err, ok := <-t.sch:
		if !ok {
			t.Errorf("%s %s (server): t.sch is closed", caller(), checkpoint)
		} else if err != nil {
			t.Errorf("%s %s (server): %v", caller(), checkpoint, err)
		}
	case <-time.After(testConnTimeout * 2):
		t.Errorf("%s %s (server): t.sch timeout", caller(), checkpoint)
	}
	if err != nil {
		t.Errorf("%s %s (client): %v", caller(), checkpoint, err)
	}
	if t.Failed() {
		t.FailNow()
	}
}
func (t *clientT) checkState(want ConnState) {
	if have := t.C.State(); have != want {
		t.Fatalf("%s C.State() expected %v; got %v", caller(), want, have)
	}
}
func (t *clientT) checkCaps(want ...string) {
	have := make([]string, 0, len(t.C.Caps))
	for v := range t.C.Caps {
		have = append(have, v)
	}
	for i := range want {
		want[i] = strings.ToUpper(want[i])
	}
	sort.Strings(have)
	sort.Strings(want)
	if !reflect.DeepEqual(have, want) {
		t.Fatalf("%s C.Caps expected %v; got %v", caller(), want, have)
	}
}
func (t *clientT) waitEOF() {
	if err := t.C.Recv(block); err != io.EOF {
		t.Fatalf("%s C.Recv() expected EOF; got %v", caller(), err)
	}
	t.checkState(Closed)
	if err := t.C.Recv(poll); err != io.EOF {
		t.Fatalf("%s C.Recv() expected EOF; got %v", caller(), err)
	}
}

func TestNewClientTimeout(T *testing.T) {
	//defer un(setLogMask(LogAll))
	newClient(T, RETURN)
}

func TestNewClientBYE(T *testing.T) {
	//defer un(setLogMask(LogAll))
	newClient(T, `S: * BYE Test server not ready, try again`+CRLF, EOF)
}

func TestNewClientOK(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T,
		`S: * OK Test server ready`+CRLF,
		`C: A1 CAPABILITY`+CRLF,
		`S: * CAPABILITY IMAP4rev1 XYZZY`+CRLF,
		`S: A1 OK Thats all she wrote!`+CRLF,
		EOF,
	)
	t.checkState(Login)
	t.checkCaps("IMAP4rev1", "XYZZY")
	t.waitEOF()

	if len(C.Data) != 1 || C.Data[0].Info != "Test server ready" {
		t.Errorf("C.Data expected greeting; got %v", C.Data)
	}
}

func TestNewClientOKCaps(T *testing.T) {
	//defer un(setLogMask(LogAll))
	_, t := newClient(T, `S: * OK [CAPABILITY IMAP4rev1] Test server ready`+CRLF, EOF)
	t.checkState(Login)
	t.checkCaps("IMAP4rev1")
	t.waitEOF()
}

func TestNewClientPREAUTH(T *testing.T) {
	//defer un(setLogMask(LogAll))
	_, t := newClient(T, `S: * PREAUTH [CAPABILITY IMAP4rev1] Test server ready`+CRLF, EOF)
	t.checkState(Auth)
	t.checkCaps("IMAP4rev1")
	t.waitEOF()
}

func TestClientBasic(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * OK [CAPABILITY IMAP4rev1 XYZZY] Test server ready`+CRLF)
	t.checkState(Login)
	t.checkCaps("IMAP4rev1", "XYZZY")

	// XYZZY
	go t.script(
		`C: A1 XYZZY`+CRLF,
		`S: A1 OK Nothing happens.`+CRLF,
	)
	cmd, err := Wait(C.Send("XYZZY"))
	t.join("XYZZY", err)

	// LOGOUT
	go t.script(
		`C: A2 LOGOUT`+CRLF,
		`S: * BYE LOGOUT Requested`+CRLF,
		`S: A2 OK Quoth the raven, nevermore...`+CRLF,
		EOF,
	)
	cmd, err = C.Logout(-1)
	t.join("LOGOUT", err)
	t.checkState(Closed)
	t.waitEOF()

	if len(cmd.Data) != 1 || cmd.Data[0].Info != "LOGOUT Requested" {
		t.Errorf("cmd.Data expected BYE; got %v", cmd.Data)
	}
	if rsp, err := cmd.Result(OK); err != nil || rsp.Info != "Quoth the raven, nevermore..." {
		t.Errorf("cmd.Result() expected OK; got %+q (%v)", rsp, err)
	}
}

func TestClientLogin(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * OK [CAPABILITY IMAP4rev1 LOGINDISABLED STARTTLS] Test server ready`+CRLF)
	t.checkCaps("IMAP4rev1", "LOGINDISABLED", "STARTTLS")

	// LOGIN should fail when LOGINDISABLED is advertised
	cmd, err := C.Login("user", "pass")
	if cmd != nil || err == nil {
		t.Fatalf("C.Login() expected error; got %#v (%v)", cmd, err)
	}

	// STARTTLS
	go t.script(
		`C: A1 STARTTLS`+CRLF,
		`S: A1 OK Begin TLS negotiation now`+CRLF,
		STARTTLS,
		`C: A2 CAPABILITY`+CRLF,
		`S: * CAPABILITY IMAP4rev1`+CRLF,
		`S: A2 OK Thats all she wrote!`+CRLF,
	)
	cmd, err = C.StartTLS(tlsConfig.client)
	t.join("STARTTLS", err)
	t.checkState(Login)
	t.checkCaps("IMAP4rev1")

	// LOGIN
	go t.script(
		`C: A3 LOGIN "user" "pass"`+CRLF,
		`S: A3 OK [CAPABILITY IMAP4rev1 COMPRESS=DEFLATE] Authenticated (Success)`+CRLF,
	)
	cmd, err = C.Login("user", "pass")
	t.join("LOGIN", err)
	t.checkState(Auth)
	t.checkCaps("IMAP4rev1", "COMPRESS=DEFLATE")

	// LOGIN should fail in Authenticated state
	cmd, err = C.Login("user", "pass")
	if cmd != nil || err == nil {
		t.Fatalf("C.Login() expected error; got %#v (%v)", cmd, err)
	}

	go t.script(EOF)
	t.join("EOF", nil)
	t.waitEOF()
}

func TestClientSelect(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * PREAUTH [CAPABILITY IMAP4rev1] Test server ready`+CRLF)
	t.checkState(Auth)

	if C.Mailbox != nil {
		t.Errorf("C.Mailbox expected nil; got\n%#v", C.Mailbox)
	}

	// EXAMINE
	go t.script(
		`C: A1 EXAMINE "INBOX"`+CRLF,
		`S: * FLAGS (\Answered \Flagged \Draft \Deleted \Seen)`+CRLF,
		`S: * OK [PERMANENTFLAGS ()] Flags permitted.`+CRLF,
		`S: * OK [UIDVALIDITY 645321] UIDs valid.`+CRLF,
		`S: * 16 EXISTS`+CRLF,
		`S: * 0 RECENT`+CRLF,
		`S: * OK [UIDNEXT 71764] Predicted next UID.`+CRLF,
		`S: A1 OK [READ-ONLY] INBOX selected. (Success)`+CRLF,
	)
	cmd, err := C.Select("INBOX", true)
	t.join("EXAMINE", err)
	t.checkState(Selected)

	if n := len(cmd.Data); n != 6 {
		t.Errorf("len(cmd.Data) expected 6; got %v", n)
	}
	status := &MailboxStatus{
		Name:        "INBOX",
		ReadOnly:    true,
		Flags:       NewFlagSet(`\Answered`, `\Flagged`, `\Draft`, `\Deleted`, `\Seen`),
		PermFlags:   NewFlagSet(),
		UIDValidity: 645321,
		Messages:    16,
		Recent:      0,
		UIDNext:     71764,
	}
	if !reflect.DeepEqual(C.Mailbox, status) {
		t.Errorf("C.Mailbox expected\n%#v; got\n%#v", status, C.Mailbox)
	}

	// Failed SELECT changes state to Auth
	go t.script(
		`C: A2 SELECT "NoSuchMailbox"`+CRLF,
		`S: A2 NO [NONEXISTENT] Unknown Mailbox: NoSuchMailbox (Failure)`+CRLF,
	)
	if cmd, err = C.Select("NoSuchMailbox", false); cmd == nil || err == nil {
		t.Fatalf("C.Select() expected NO; got %#v (%v)", cmd, err)
	}
	t.join("NoSuchMailbox", nil)
	t.checkState(Auth)

	if rsp, err := cmd.Result(NO); err != nil || rsp.Info != "Unknown Mailbox: NoSuchMailbox (Failure)" {
		t.Errorf("cmd.Result() expected NO; got %+q (%v)", rsp, err)
	}
	if C.Mailbox != nil {
		t.Errorf("C.Mailbox expected nil; got\n%#v", C.Mailbox)
	}

	// SELECT
	go t.script(
		`C: A3 SELECT "INBOX"`+CRLF,
		`S: * 172 EXISTS`+CRLF,
		`S: * 1 RECENT`+CRLF,
		`S: * OK [UNSEEN 12] Message 12 is first unseen`+CRLF,
		`S: * OK [UIDVALIDITY 3857529045] UIDs valid`+CRLF,
		`S: * OK [UIDNEXT 4392] Predicted next UID`+CRLF,
		`S: * FLAGS (\Answered \Flagged \Deleted \Seen \Draft)`+CRLF,
		`S: * OK [PERMANENTFLAGS (\Deleted \Seen \*)] Limited`+CRLF,
		`S: A3 OK [READ-WRITE] SELECT completed`+CRLF,
	)
	cmd, err = C.Select("INBOX", false)
	t.join("SELECT", err)
	t.checkState(Selected)

	if n := len(cmd.Data); n != 7 {
		t.Errorf("len(cmd.Data) expected 7; got %v", n)
	}
	status = &MailboxStatus{
		Name:        "INBOX",
		Messages:    172,
		Recent:      1,
		Unseen:      12,
		UIDValidity: 3857529045,
		UIDNext:     4392,
		Flags:       NewFlagSet(`\Answered`, `\Flagged`, `\Deleted`, `\Seen`, `\Draft`),
		PermFlags:   NewFlagSet(`\Deleted`, `\Seen`, `\*`),
	}
	if !reflect.DeepEqual(C.Mailbox, status) {
		t.Errorf("C.Mailbox expected\n%#v; got\n%#v", status, C.Mailbox)
	}

	// RESELECT from Selected state
	go t.script(
		`C: A4 SELECT "funny"`+CRLF,
		`S: * 1 EXISTS`+CRLF,
		`S: * 1 RECENT`+CRLF,
		`S: * OK [UNSEEN 1] Message 1 is first unseen`+CRLF,
		`S: * OK [UIDVALIDITY 3857529045] Validity session-only`+CRLF,
		`S: * OK [UIDNEXT 2] Predicted next UID`+CRLF,
		`S: * NO [UIDNOTSTICKY] Non-persistent UIDs`+CRLF,
		`S: * FLAGS (\Answered \Flagged \Deleted \Seen \Draft)`+CRLF,
		`S: * OK [PERMANENTFLAGS (\Deleted \Seen)] Limited`+CRLF,
		`S: A4 OK [READ-WRITE] SELECT completed`+CRLF,
	)
	cmd, err = C.Select("funny", false)
	t.join("RESELECT", err)
	t.checkState(Selected)

	if n := len(cmd.Data); n != 8 {
		t.Errorf("len(cmd.Data) expected 8; got %v", n)
	}
	status = &MailboxStatus{
		Name:         "funny",
		Messages:     1,
		Recent:       1,
		Unseen:       1,
		UIDValidity:  3857529045,
		UIDNext:      2,
		UIDNotSticky: true,
		Flags:        NewFlagSet(`\Answered`, `\Flagged`, `\Deleted`, `\Seen`, `\Draft`),
		PermFlags:    NewFlagSet(`\Deleted`, `\Seen`),
	}
	if !reflect.DeepEqual(C.Mailbox, status) {
		t.Errorf("C.Mailbox expected\n%#v; got\n%#v", status, C.Mailbox)
	}

	go t.script(EOF)
	t.join("EOF", nil)
	t.waitEOF()
}

func TestClientMulti1(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * PREAUTH [CAPABILITY IMAP4rev1] Test server ready`+CRLF)

	go t.script(
		`C: A1 LIST "" "*"`+CRLF,
		`S: * LIST () "/" INBOX`+CRLF,
		`C: A2 LSUB "" "*"`+CRLF,
		`S: * LIST () "/" blurdybloop`+CRLF,
		`S: * LSUB () "/" INBOX`+CRLF,
		`S: A1 OK LIST completed`+CRLF,
		`S: A2 OK LSUB completed`+CRLF,
		EOF,
	)
	cmd1, err := C.List("", "*")
	if err != nil {
		t.Fatalf("C.List() unexpected error; %v", err)
	}
	cmd2, err := Wait(C.LSub("", "*"))
	t.join("LIST/LSUB", err)

	if n := len(cmd1.Data); n != 2 {
		t.Errorf("len(cmd1.Data) expected 2; got %v", n)
	}
	if n := len(cmd2.Data); n != 1 {
		t.Errorf("len(cmd2.Data) expected 1; got %v", n)
	}
	if cmd1.InProgress() {
		t.Errorf("cmd1.InProgress() expected false; got true")
	}
	t.waitEOF()
}

func TestClientMulti2(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * PREAUTH [CAPABILITY IMAP4rev1] Test server ready`+CRLF)

	go t.script(
		`C: A1 LIST "" "*"`+CRLF,
		`S: * LIST () "/" INBOX`+CRLF,
		`C: A2 LSUB "" {1}`+CRLF,
		`S: + Ready for additional command text`+CRLF,
		`C: *`,
		`C: `+CRLF,
		`S: * LSUB () "/" INBOX`+CRLF,
		`S: * LIST () "/" blurdybloop`+CRLF,
		`S: A2 OK LSUB completed`+CRLF,
		`S: A1 OK LIST completed`+CRLF,
		EOF,
	)
	cmd1, err := C.List("", "*")
	if err != nil {
		t.Fatalf("C.List() unexpected error; %v", err)
	}
	cmd2, err := Wait(C.Send("LSUB", `""`, lit("*")))
	t.join("LIST/LSUB", err)

	if n := len(cmd1.Data); n != 2 {
		t.Errorf("len(cmd1.Data) expected 2; got %v", n)
	}
	if n := len(cmd2.Data); n != 1 {
		t.Errorf("len(cmd2.Data) expected 1; got %v", n)
	}
	if !cmd1.InProgress() {
		t.Errorf("cmd1.InProgress() expected true; got false")
	} else if err = C.Recv(block); err != nil {
		t.Errorf("C.Recv() unexpected error; %v", err)
	} else if cmd1.InProgress() {
		t.Errorf("cmd1.InProgress() expected false; got true")
	}
	t.waitEOF()
}

func TestClientMulti3(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * PREAUTH [CAPABILITY IMAP4rev1] Test server ready`+CRLF)

	go t.script(
		`C: A1 LIST "" "*"`+CRLF,
		`C: A2 LSUB "" {1}`+CRLF,
		`S: * LIST () "/" INBOX`+CRLF,
		`S: + Ready for additional command text`+CRLF,
		`C: *`,
		`C: `+CRLF,
		`S: * LSUB () "/" INBOX`+CRLF,
		`C: A3 STATUS {11}`+CRLF,
		`S: * LIST () "/" blurdybloop`+CRLF,
		`S: + Ready for additional command text`+CRLF,
		`C: blurdybloop`,
		`C:  (UIDNEXT MESSAGES)`+CRLF,
		`S: * LSUB () "/" blurdybloop`+CRLF,
		`S: A2 OK LSUB completed`+CRLF,
		`S: * LIST () "/" funny`+CRLF,
		`S: * STATUS blurdybloop (MESSAGES 231 UIDNEXT 44292)`+CRLF,
		`S: A1 OK LIST completed`+CRLF,
		`S: A3 OK STATUS completed`+CRLF,
		EOF,
	)
	cmd1, err := C.List("", "*")
	if err != nil {
		t.Fatalf("C.List() unexpected error; %v", err)
	}
	cmd2, err := C.Send("LSUB", `""`, lit("*"))
	if err != nil {
		t.Fatalf("C.LSub() unexpected error; %v", err)
	}
	cmd3, err := Wait(C.Send("STATUS", lit("blurdybloop"), []Field{"UIDNEXT", "MESSAGES"}))
	if err != nil {
		t.Fatalf("C.Status() unexpected error; %v", err)
	}
	t.join("LIST/LSUB/STATUS", err)

	if n := len(cmd1.Data); n != 3 {
		t.Errorf("len(cmd1.Data) expected 3; got %v", n)
	}
	if n := len(cmd2.Data); n != 2 {
		t.Errorf("len(cmd2.Data) expected 2; got %v", n)
	}
	if n := len(cmd3.Data); n != 1 {
		t.Errorf("len(cmd3.Data) expected 1; got %v", n)
	} else {
		have := cmd3.Data[0].MailboxStatus()
		want := &MailboxStatus{
			Name:     "blurdybloop",
			Messages: 231,
			UIDNext:  44292,
		}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("MailboxStatus() expected\n%#v; got\n%#v", want, have)
		}
	}
	if cmd1.InProgress() || cmd2.InProgress() || cmd3.InProgress() {
		t.Errorf("cmd{1,2,3}.InProgress() expected false; got true")
	} else {
		if rsp, err := cmd1.Result(OK); err != nil || rsp.Info != "LIST completed" {
			t.Errorf("cmd1.Result() expected OK; got %+q (%v)", rsp, err)
		}
		if rsp, err := cmd2.Result(OK); err != nil || rsp.Info != "LSUB completed" {
			t.Errorf("cmd2.Result() expected OK; got %+q (%v)", rsp, err)
		}
	}
	t.waitEOF()
}

func TestClientAuthPlain(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * OK [CAPABILITY IMAP4rev1 STARTTLS AUTH=PLAIN] Test server ready`+CRLF)

	// AUTH=PLAIN should fail when the connection is not encrypted
	cmd, err := C.Auth(PlainAuth("test", "test", "test"))
	if cmd != nil || err == nil {
		t.Fatalf("C.Auth(PLAIN) expected error; got %#v (%v)", cmd, err)
	}

	// STARTTLS
	go t.script(
		`C: A1 STARTTLS`+CRLF,
		`S: A1 OK Begin TLS negotiation now`+CRLF,
		STARTTLS,
		`C: A2 CAPABILITY`+CRLF,
		`S: * CAPABILITY IMAP4rev1 AUTH=PLAIN`+CRLF,
		`S: A2 OK Thats all she wrote!`+CRLF,
	)
	cmd, err = C.StartTLS(tlsConfig.client)
	t.join("STARTTLS", err)
	t.checkState(Login)
	t.checkCaps("IMAP4rev1", "AUTH=PLAIN")

	// AUTH=PLAIN
	go t.script(
		`C: A3 AUTHENTICATE PLAIN`+CRLF,
		`S: + `+CRLF,
		`C: dGVzdAB0ZXN0AHRlc3Q=`+CRLF,
		`S: A3 OK Success`+CRLF,
		`C: A4 CAPABILITY`+CRLF,
		`S: * CAPABILITY IMAP4rev1`+CRLF,
		`S: A4 OK Thats all she wrote!`+CRLF,
		EOF,
	)
	cmd, err = C.Auth(PlainAuth("test", "test", "test"))
	t.join("AUTH=PLAIN", err)
	t.checkState(Auth)
	t.checkCaps("IMAP4rev1")
	t.waitEOF()
}

func TestClientAuthExternal1(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * OK [CAPABILITY IMAP4rev1 AUTH=EXTERNAL] Test server ready`+CRLF)

	go t.script(
		`C: A1 AUTHENTICATE EXTERNAL`+CRLF,
		`S: + `+CRLF,
		`C: `+CRLF,
		`S: A1 OK [CAPABILITY IMAP4rev1] Success`+CRLF,
		EOF,
	)
	_, err := C.Auth(ExternalAuth(""))
	t.join("AUTH=EXTERNAL", err)
	t.checkState(Auth)
	t.checkCaps("IMAP4rev1")
	t.waitEOF()
}

func TestClientAuthExternal2(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * OK [CAPABILITY IMAP4rev1 AUTH=EXTERNAL SASL-IR] Test server ready`+CRLF)

	go t.script(
		`C: A1 AUTHENTICATE EXTERNAL =`+CRLF,
		`S: A1 OK [CAPABILITY IMAP4rev1] Success`+CRLF,
		EOF,
	)
	_, err := C.Auth(ExternalAuth(""))
	t.join("AUTH=EXTERNAL", err)
	t.checkState(Auth)
	t.checkCaps("IMAP4rev1")
	t.waitEOF()
}

func TestClientAuthExternal3(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * OK [CAPABILITY IMAP4rev1 AUTH=EXTERNAL SASL-IR] Test server ready`+CRLF)

	go t.script(
		`C: A1 AUTHENTICATE EXTERNAL dGVzdA==`+CRLF,
		`S: A1 OK [CAPABILITY IMAP4rev1] Success`+CRLF,
		EOF,
	)
	_, err := C.Auth(ExternalAuth("test"))
	t.join("AUTH=EXTERNAL", err)
	t.checkState(Auth)
	t.checkCaps("IMAP4rev1")
	t.waitEOF()
}

func TestClientClose1(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * PREAUTH [CAPABILITY IMAP4rev1] Test server ready`+CRLF)

	// EXAMINE
	go t.script(
		`C: A1 EXAMINE "INBOX"`+CRLF,
		`S: * FLAGS (\Answered \Flagged \Draft \Deleted \Seen)`+CRLF,
		`S: * OK [PERMANENTFLAGS ()] Flags permitted.`+CRLF,
		`S: * OK [UIDVALIDITY 645321] UIDs valid.`+CRLF,
		`S: * 16 EXISTS`+CRLF,
		`S: * 0 RECENT`+CRLF,
		`S: * OK [UIDNEXT 71764] Predicted next UID.`+CRLF,
		`S: A1 OK [READ-ONLY] INBOX selected. (Success)`+CRLF,
	)
	_, err := C.Select("INBOX", true)
	t.join("EXAMINE", err)
	t.checkState(Selected)

	// CLOSE
	go t.script(
		`C: A2 EXAMINE "GOIMAPAAAAAA"`+CRLF,
		`S: A2 NO Nonexitent mailbox`+CRLF,
		EOF,
	)
	_, err = C.Close(false)
	t.join("CLOSE", err)
	t.checkState(Auth)
	t.waitEOF()
}

func TestClientClose2(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * PREAUTH [CAPABILITY IMAP4rev1 UNSELECT] Test server ready`+CRLF)

	// EXAMINE
	go t.script(
		`C: A1 EXAMINE "INBOX"`+CRLF,
		`S: * FLAGS (\Answered \Flagged \Draft \Deleted \Seen)`+CRLF,
		`S: * OK [PERMANENTFLAGS ()] Flags permitted.`+CRLF,
		`S: * OK [UIDVALIDITY 645321] UIDs valid.`+CRLF,
		`S: * 16 EXISTS`+CRLF,
		`S: * 0 RECENT`+CRLF,
		`S: * OK [UIDNEXT 71764] Predicted next UID.`+CRLF,
		`S: A1 OK [READ-ONLY] INBOX selected. (Success)`+CRLF,
	)
	_, err := C.Select("INBOX", true)
	t.join("EXAMINE", err)
	t.checkState(Selected)

	// CLOSE
	go t.script(
		`C: A2 UNSELECT`+CRLF,
		`S: A2 OK Unselect completed`+CRLF,
		EOF,
	)
	_, err = C.Close(false)
	t.join("CLOSE", err)
	t.checkState(Auth)
	t.waitEOF()
}

func TestClientIdle(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * PREAUTH [CAPABILITY IMAP4rev1 IDLE] Test server ready`+CRLF)

	// IDLE
	go t.script(
		`C: A1 IDLE`+CRLF,
		`S: + idling`+CRLF,
	)
	cmd1, err := C.Idle()
	t.join("IDLE", err)
	C.Data = nil

	// UPDATE
	go t.script(
		`S: * 4 EXISTS`+CRLF,
		`S: * 2 EXPUNGE`+CRLF,
		`S: * 3 EXISTS`+CRLF,
	)
	for i := 0; i < 3; i++ {
		if err = C.Recv(block); err != nil {
			t.Fatalf("C.Recv() unexpected error; %v", err)
		} else if len(C.Data) != i+1 {
			t.Fatalf("len(C.Data) expected %d; got %d", i+1, len(C.Data))
		}
		switch i {
		case 0:
			if C.Data[i].Label != "EXISTS" || C.Data[i].Value() != 4 {
				t.Fatalf("C.Data[%d] expected 4 EXISTS; got %v", i, C.Data[i])
			}
		case 1:
			if C.Data[i].Label != "EXPUNGE" || C.Data[i].Value() != 2 {
				t.Fatalf("C.Data[%d] expected 2 EXPUNGE; got %v", i, C.Data[i])
			}
		case 2:
			if C.Data[i].Label != "EXISTS" || C.Data[i].Value() != 3 {
				t.Fatalf("C.Data[%d] expected 3 EXISTS; got %v", i, C.Data[i])
			}
		}
	}
	t.join("UPDATE", err)

	// DONE
	go t.script(
		`C: DONE`+CRLF,
		`S: A1 OK IDLE terminated`+CRLF,
		EOF,
	)
	cmd2, err := C.IdleTerm()
	t.join("DONE", err)
	t.waitEOF()

	if cmd1 != cmd2 {
		t.Fatal("cmd1 == cmd2 expected true; got false")
	}
}

func TestClientQuota(T *testing.T) {
	//defer un(setLogMask(LogAll))
	C, t := newClient(T, `S: * PREAUTH [CAPABILITY IMAP4rev1 QUOTA] Test server ready`+CRLF)

	// SETQUOTA1
	go t.script(
		`C: A1 SETQUOTA "" (STORAGE 512)`+CRLF,
		`S: * QUOTA "" (STORAGE 10 512)`+CRLF,
		`S: A1 OK Setquota completed`+CRLF,
	)
	_, err := Wait(C.SetQuota("", &Quota{"STORAGE", 0, 512}))
	t.join("SETQUOTA1", err)

	// SETQUOTA2
	go t.script(
		`C: A2 SETQUOTA "" (STORAGE 512 MESSAGE 100)`+CRLF,
		`S: * QUOTA "" (STORAGE 10 512 MESSAGE 20 100)`+CRLF,
		`S: A2 OK Setquota completed`+CRLF,
	)
	_, err = Wait(C.SetQuota("", &Quota{"STORAGE", 0, 512}, &Quota{"MESSAGE", 0, 100}))
	t.join("SETQUOTA2", err)

	// GETQUOTA
	go t.script(
		`C: A3 GETQUOTA ""`+CRLF,
		`S: * QUOTA "" (STORAGE 10 512)`+CRLF,
		`S: A3 OK Getquota completed`+CRLF,
	)
	_, err = Wait(C.GetQuota(""))
	t.join("GETQUOTA", err)

	// GETQUOTAROOT
	go t.script(
		`C: A4 GETQUOTAROOT "INBOX"`+CRLF,
		`S: * QUOTAROOT INBOX ""`+CRLF,
		`S: * QUOTA "" (STORAGE 10 512)`+CRLF,
		`S: A4 OK Getquota completed`+CRLF,
		EOF,
	)
	_, err = Wait(C.GetQuotaRoot("INBOX"))
	t.join("GETQUOTAROOT", err)
	t.waitEOF()
}
