// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"crypto/tls"
	"io"
	"net"
	"time"
)

// Timeout values for the Dial functions.
const (
	netTimeout    = 30 * time.Second // Time to establish a TCP connection
	clientTimeout = 60 * time.Second // Time to receive greeting and capabilities
)

// Dial returns a new Client connected to an IMAP server at addr.
func Dial(addr string) (c *Client, err error) {
	addr = defaultPort(addr, "143")
	conn, err := net.DialTimeout("tcp", addr, netTimeout)
	if err == nil {
		host, _, _ := net.SplitHostPort(addr)
		if c, err = NewClient(conn, host, clientTimeout); err != nil {
			conn.Close()
		}
	}
	return
}

// DialTLS returns a new Client connected to an IMAP server at addr using the
// specified config for encryption.
func DialTLS(addr string, config *tls.Config) (c *Client, err error) {
	addr = defaultPort(addr, "993")
	conn, err := net.DialTimeout("tcp", addr, netTimeout)
	if err == nil {
		host, _, _ := net.SplitHostPort(addr)
		tlsConn := tls.Client(conn, setServerName(config, host))
		if c, err = NewClient(tlsConn, host, clientTimeout); err != nil {
			conn.Close()
		}
	}
	return
}

// Wait is a convenience function for transforming asynchronous commands into
// synchronous ones. The error is nil if and only if the command is completed
// with OK status condition. Usage example:
//
// 	cmd, err := imap.Wait(c.Fetch(...))
func Wait(cmd *Command, err error) (*Command, error) {
	if err == nil {
		_, err = cmd.Result(OK)
	}
	return cmd, err
}

// Capability requests a listing of capabilities supported by the server. The
// client automatically requests capabilities when the connection is first
// established, after a successful STARTTLS command, and after user
// authentication, making it unnecessary to call this method directly in most
// cases. The current capabilities are available in c.Caps.
//
// This command is synchronous.
func (c *Client) Capability() (cmd *Command, err error) {
	return Wait(c.Send("CAPABILITY"))
}

// Noop does nothing, but it allows the server to send status updates, which are
// delivered to the unilateral server data queue (c.Data). It can also be used
// to reset any inactivity autologout timer on the server.
func (c *Client) Noop() (cmd *Command, err error) {
	return c.Send("NOOP")
}

// Logout informs the server that the client is done with the connection. This
// method must be called to close the connection and free all client resources.
//
// A negative timeout allows the client to wait indefinitely for the normal
// logout sequence to complete. A timeout of 0 causes the connection to be
// closed immediately without actually sending the LOGOUT command. A positive
// timeout behaves as expected, returning ErrTimeout if the normal logout
// sequence is not completed in the allocated time. The connection is always
// closed when this method returns.
//
// This command is synchronous.
func (c *Client) Logout(timeout time.Duration) (cmd *Command, err error) {
	if c.state == Closed {
		return nil, ErrNotAllowed
	}
	defer c.setState(Closed)
	defer c.close("logout error")

	c.setState(Logout)
	if timeout == 0 {
		err = c.close("immediate logout")
	} else {
		if timeout > 0 {
			c.t.conn.SetDeadline(time.Now().Add(timeout))
		}
		cmd, err = Wait(c.Send("LOGOUT"))
	}
	for err == nil {
		if err = c.Recv(block); err == io.EOF {
			return cmd, nil
		}
	}
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		err = ErrTimeout
	}
	return
}

// StartTLS enables session privacy protection and integrity checking. The
// server must advertise STARTTLS capability for this command to be available.
// The client automatically requests new capabilities if the TLS handshake is
// successful.
//
// This command is synchronous.
func (c *Client) StartTLS(config *tls.Config) (cmd *Command, err error) {
	if !c.Caps["STARTTLS"] {
		return nil, NotAvailableError("STARTTLS")
	} else if c.t.Encrypted() {
		return nil, ErrEncryptionActive
	}
	if cmd, err = Wait(c.Send("STARTTLS")); err == nil {
		if c.rch != nil {
			// Should never happen
			panic("imap: receiver is active, cannot perform TLS handshake")
		}
		if err = c.t.EnableTLS(setServerName(config, c.host)); err == nil {
			_, err = c.Capability()
		}
	}
	return
}

// Auth performs SASL challenge-response authentication. The client
// automatically requests new capabilities if authentication is successful.
//
// This command is synchronous.
func (c *Client) Auth(a SASL) (cmd *Command, err error) {
	info := ServerInfo{c.host, c.t.Encrypted(), c.getCaps("AUTH=")}
	mech, cr, err := a.Start(&info)
	if err != nil {
		return
	} else if name := "AUTH=" + mech; !c.Caps[name] {
		return nil, NotAvailableError(name)
	}
	args := []Field{mech, nil}[:1]

	// Initial response is sent with the command if the server supports SASL-IR
	if cr = b64enc(cr); cr != nil && c.Caps["SASL-IR"] {
		if len(cr) > 0 {
			args = append(args, cr)
		} else {
			args = append(args, "=")
		}
		cr = nil
	}
	cmd, err = c.Send("AUTHENTICATE", args...)

	// Challenge-response loop
	var rsp *Response
	var abort error
	for err == nil && cmd.InProgress() {
		rsp, err = c.checkContinue(cmd, true)
		if err == nil && rsp.Type == Continue {
			if cr == nil {
				if cr, abort = a.Next(rsp.Challenge()); abort == nil {
					cr = b64enc(cr)
				} else if err = c.t.WriteLine([]byte("*")); err == nil {
					err = c.t.Flush()
					break
				}
			}
			err, cr = c.t.WriteLine(cr), nil
		}
	}

	// Wait for command completion
	if err == nil {
		if rsp, err = cmd.Result(OK); err == nil {
			c.setState(Auth)
			if rsp.Label != "CAPABILITY" {
				_, err = c.Capability()
			}
		} else if abort != nil && rsp != nil && rsp.Status == BAD {
			err = abort
		}
	}
	return
}

// Login performs plaintext username/password authentication. This command is
// disabled when the server advertises LOGINDISABLED capability. The client
// automatically requests new capabilities if authentication is successful.
//
// This command is synchronous.
func (c *Client) Login(username, password string) (cmd *Command, err error) {
	if c.Caps["LOGINDISABLED"] {
		return nil, NotAvailableError("LOGIN")
	}
	cmd, err = Wait(c.Send("LOGIN", c.Quote(username), c.Quote(password)))
	if err == nil {
		c.setState(Auth)
		if cmd.result.Label != "CAPABILITY" {
			// Gmail servers send an untagged CAPABILITY response after
			// successful authentication. RFC 3501 states that the CAPABILITY
			// response code in command completion should be used instead, so we
			// ignore the untagged response.
			_, err = c.Capability()
		}
	}
	return
}

// Select opens a mailbox on the server for read-write or read-only access. The
// EXAMINE command is used when readonly is set to true. However, even when
// readonly is false, the server may decide not to give read-write access. The
// server may also change access while the mailbox is open. The current mailbox
// status is available from c.Mailbox while the client is in the Selected state.
//
// This command is synchronous.
func (c *Client) Select(mbox string, readonly bool) (cmd *Command, err error) {
	return Wait(c.doSelect(mbox, readonly))
}

// Create creates a new mailbox on the server.
func (c *Client) Create(mbox string) (cmd *Command, err error) {
	return c.Send("CREATE", c.Quote(UTF7Encode(mbox)))
}

// Delete permanently removes a mailbox and all of its contents from the server.
func (c *Client) Delete(mbox string) (cmd *Command, err error) {
	return c.Send("DELETE", c.Quote(UTF7Encode(mbox)))
}

// Rename changes the name of a mailbox.
func (c *Client) Rename(old, new string) (cmd *Command, err error) {
	return c.Send("RENAME", c.Quote(UTF7Encode(old)), c.Quote(UTF7Encode(new)))
}

// Subscribe adds the specified mailbox name to the server's set of "active" or
// "subscribed" mailboxes as returned by the LSUB command.
func (c *Client) Subscribe(mbox string) (cmd *Command, err error) {
	return c.Send("SUBSCRIBE", c.Quote(UTF7Encode(mbox)))
}

// Unsubscribe removes the specified mailbox name from the server's set of
// "active" or "subscribed" mailboxes as returned by the LSUB command.
func (c *Client) Unsubscribe(mbox string) (cmd *Command, err error) {
	return c.Send("UNSUBSCRIBE", c.Quote(UTF7Encode(mbox)))
}

// List returns a subset of mailbox names from the complete set of all names
// available to the client.
//
// See RFC 3501 sections 6.3.8 and 7.2.2, and RFC 2683 for detailed information
// about the LIST and LSUB commands.
func (c *Client) List(ref, mbox string) (cmd *Command, err error) {
	return c.Send("LIST", c.Quote(ref), c.Quote(mbox))
}

// LSub returns a subset of mailbox names from the set of names that the user
// has declared as being "active" or "subscribed".
func (c *Client) LSub(ref, mbox string) (cmd *Command, err error) {
	return c.Send("LSUB", c.Quote(ref), c.Quote(mbox))
}

// Status requests the status of the indicated mailbox. The currently defined
// status data items that can be requested are: MESSAGES, RECENT, UIDNEXT,
// UIDVALIDITY, and UNSEEN. All data items are requested by default.
func (c *Client) Status(mbox string, items ...string) (cmd *Command, err error) {
	var f []Field
	if len(items) == 0 {
		f = []Field{"MESSAGES", "RECENT", "UIDNEXT", "UIDVALIDITY", "UNSEEN"}
	} else {
		f = stringsToFields(items)
	}
	return c.Send("STATUS", c.Quote(UTF7Encode(mbox)), f)
}

// Append appends the literal argument as a new message to the end of the
// specified destination mailbox. Flags and internal date arguments are optional
// and may be set to nil.
func (c *Client) Append(mbox string, flags FlagSet, idate *time.Time, msg Literal) (cmd *Command, err error) {
	f := []Field{c.Quote(UTF7Encode(mbox)), nil, nil, nil}[:1]
	if flags != nil {
		f = append(f, flags)
	}
	if idate != nil {
		f = append(f, *idate)
	}
	return c.Send("APPEND", append(f, msg)...)
}

// Check requests a checkpoint of the currently selected mailbox. A checkpoint
// is an implementation detail of the server and may be equivalent to a NOOP.
func (c *Client) Check() (cmd *Command, err error) {
	return c.Send("CHECK")
}

// Close closes the currently selected mailbox, returning the client to the
// authenticated state. If expunge is true, all messages marked for deletion are
// permanently removed from the mailbox.
//
// If expunge is false and UNSELECT capability is not advertised, the client
// issues the EXAMINE command with a non-existent mailbox name. This closes the
// current mailbox without expunging it, but the "successful" command completion
// status will be NO instead of OK.
//
// This command is synchronous.
func (c *Client) Close(expunge bool) (cmd *Command, err error) {
	name := "CLOSE"
	if !expunge {
		if !c.Caps["UNSELECT"] {
			mbox := "GOIMAP" + randStr(6)
			if cmd, err = c.doSelect(mbox, true); err == nil {
				_, err = cmd.Result(NO)
			}
			return
		}
		name = "UNSELECT"
	}
	if cmd, err = Wait(c.Send(name)); err == nil {
		c.setState(Auth)
	}
	return
}

// Expunge permanently removes all messages that have the \Deleted flag set from
// the currently selected mailbox. If UIDPLUS capability is advertised, the
// operation can be restricted to messages with specific UIDs by specifying a
// non-nil uids argument.
func (c *Client) Expunge(uids *SeqSet) (cmd *Command, err error) {
	if uids != nil {
		if !c.Caps["UIDPLUS"] {
			return nil, NotAvailableError("UIDPLUS")
		}
		return c.Send("UID EXPUNGE", uids)
	}
	return c.Send("EXPUNGE")
}

// Search searches the mailbox for messages that match the given searching
// criteria. See RFC 3501 section 6.4.4 for a list of all valid search keys. It
// is the caller's responsibility to quote strings when necessary. All strings
// must use UTF-8 encoding.
func (c *Client) Search(spec ...Field) (cmd *Command, err error) {
	return c.Send("SEARCH", append([]Field{"CHARSET", "UTF-8"}, spec...)...)
}

// Fetch retrieves data associated with the specified message(s) in the mailbox.
// See RFC 3501 section 6.4.5 for a list of all valid message data items and
// macros.
func (c *Client) Fetch(seq *SeqSet, items ...string) (cmd *Command, err error) {
	return c.Send("FETCH", seq, stringsToFields(items))
}

// Store alters data associated with the specified message(s) in the mailbox.
func (c *Client) Store(seq *SeqSet, item string, value Field) (cmd *Command, err error) {
	return c.Send("STORE", seq, item, value)
}

// Copy copies the specified message(s) to the end of the specified destination
// mailbox.
func (c *Client) Copy(seq *SeqSet, mbox string) (cmd *Command, err error) {
	return c.Send("COPY", seq, c.Quote(UTF7Encode(mbox)))
}

// Move moves the specified message(s) to the end of the specified destination
// mailbox.
func (c *Client) Move(seq *SeqSet, mbox string) (cmd *Command, err error) {
	return c.Send("MOVE", seq, c.Quote(UTF7Encode(mbox)))
}

// UIDSearch is identical to Search, but the numbers returned in the response
// are unique identifiers instead of message sequence numbers.
func (c *Client) UIDSearch(spec ...Field) (cmd *Command, err error) {
	return c.Send("UID SEARCH", append([]Field{"CHARSET", "UTF-8"}, spec...)...)
}

// UIDFetch is identical to Fetch, but the seq argument is interpreted as
// containing unique identifiers instead of message sequence numbers.
func (c *Client) UIDFetch(seq *SeqSet, items ...string) (cmd *Command, err error) {
	return c.Send("UID FETCH", seq, stringsToFields(items))
}

// UIDStore is identical to Store, but the seq argument is interpreted as
// containing unique identifiers instead of message sequence numbers.
func (c *Client) UIDStore(seq *SeqSet, item string, value Field) (cmd *Command, err error) {
	return c.Send("UID STORE", seq, item, value)
}

// UIDCopy is identical to Copy, but the seq argument is interpreted as
// containing unique identifiers instead of message sequence numbers.
func (c *Client) UIDCopy(seq *SeqSet, mbox string) (cmd *Command, err error) {
	return c.Send("UID COPY", seq, c.Quote(UTF7Encode(mbox)))
}

// UIDMove is identical to Move, but the seq argument is interpreted as
// containing unique identifiers instead of message sequence numbers.
func (c *Client) UIDMove(seq *SeqSet, mbox string) (cmd *Command, err error) {
	return c.Send("UID MOVE", seq, c.Quote(UTF7Encode(mbox)))
}

// SetQuota changes the resource limits of the specified quota root. See RFC
// 2087 for additional information.
func (c *Client) SetQuota(root string, quota ...*Quota) (cmd *Command, err error) {
	if !c.Caps["QUOTA"] {
		return nil, NotAvailableError("QUOTA")
	}
	f := make([]Field, 0, len(quota)*2)
	for _, q := range quota {
		f = append(f, q.Resource, q.Limit)
	}
	return c.Send("SETQUOTA", c.Quote(root), f)
}

// GetQuota returns the quota root's resource usage and limits. See RFC 2087 for
// additional information.
func (c *Client) GetQuota(root string, quota ...*Quota) (cmd *Command, err error) {
	if !c.Caps["QUOTA"] {
		return nil, NotAvailableError("QUOTA")
	}
	return c.Send("GETQUOTA", c.Quote(root))
}

// GetQuotaRoot returns the list of quota roots for the specified mailbox, and
// the resource usage and limits for each quota root. See RFC 2087 for
// additional information.
func (c *Client) GetQuotaRoot(mbox string) (cmd *Command, err error) {
	if !c.Caps["QUOTA"] {
		return nil, NotAvailableError("QUOTA")
	}
	return c.Send("GETQUOTAROOT", c.Quote(UTF7Encode(mbox)))
}

// Idle places the client into an idle state where the server is free to send
// unsolicited mailbox update messages. No other commands are allowed to run
// while the client is idling. Use c.IdleTerm to terminate the command. See RFC
// 2177 for additional information.
func (c *Client) Idle() (cmd *Command, err error) {
	if !c.Caps["IDLE"] {
		return nil, NotAvailableError("IDLE")
	}
	if cmd, err = c.Send("IDLE"); err == nil {
		var rsp *Response
		if rsp, err = c.checkContinue(cmd, true); err == nil {
			if rsp.Type == Continue {
				c.Logln(LogState, "Client is idling...")
			} else {
				_, err = cmd.Result(OK)
			}
		}
	}
	return
}

// IdleTerm terminates the IDLE command. It returns the same Command instance as
// the original Idle call.
func (c *Client) IdleTerm() (cmd *Command, err error) {
	if len(c.tags) == 1 {
		if cmd = c.cmds[c.tags[0]]; cmd.name == "IDLE" {
			if err = c.t.WriteLine([]byte("DONE")); err == nil {
				if err = c.t.Flush(); err == nil {
					_, err = cmd.Result(OK)
					c.Logln(LogState, "Client is done idling")
				}
			}
		}
	}
	return
}

// ID provides client identification information to the server. See RFC 2971 for
// additional information.
func (c *Client) ID(info ...string) (cmd *Command, err error) {
	if !c.Caps["ID"] {
		return nil, NotAvailableError("ID")
	}
	f := make([]Field, len(info))
	for i, v := range info {
		f[i] = c.Quote(v)
	}
	return c.Send("ID", f)
}

// CompressDeflate enables data compression using the DEFLATE algorithm. The
// compression level must be between -1 and 9 (see compress/flate). See RFC 4978
// for additional information.
//
// This command is synchronous.
func (c *Client) CompressDeflate(level int) (cmd *Command, err error) {
	if !c.Caps["COMPRESS=DEFLATE"] {
		return nil, NotAvailableError("COMPRESS=DEFLATE")
	} else if c.t.Compressed() {
		return nil, ErrCompressionActive
	}
	if cmd, err = Wait(c.Send("COMPRESS", "DEFLATE")); err == nil {
		err = c.t.EnableDeflate(level)
	}
	return
}

// Enable takes a list of capability names and requests the server to enable the
// named extensions. See RFC 5161 for additional information.
//
// This command is synchronous.
func (c *Client) Enable(caps ...string) (cmd *Command, err error) {
	return Wait(c.Send("ENABLE", stringsToFields(caps)))
}

// doSelect opens the specified mailbox, returning an error if the command
// completion status is other than OK or NO.
func (c *Client) doSelect(mbox string, readonly bool) (cmd *Command, err error) {
	name := "SELECT"
	if readonly {
		name = "EXAMINE"
	}
	if cmd, err = c.Send(name, c.Quote(UTF7Encode(mbox))); err == nil {
		prev := c.Mailbox
		c.setState(Auth)
		c.Mailbox = newMailboxStatus(mbox)

		var rsp *Response
		if rsp, err = cmd.Result(OK | NO); err == nil {
			if rsp.Status == OK {
				c.setState(Selected)
			} else {
				c.Mailbox = nil
			}
		} else if c.Mailbox = prev; prev != nil && c.state == Auth {
			c.setState(Selected)
		}
	}
	return
}

// stringsToFields converts []string to []Field.
func stringsToFields(s []string) []Field {
	f := make([]Field, len(s))
	for i, v := range s {
		f[i] = v
	}
	return f
}
