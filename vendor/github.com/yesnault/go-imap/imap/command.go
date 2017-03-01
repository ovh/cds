// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// ErrAborted is returned by Command.Result when the command execution is
// interrupted prior to receiving a completion response from the server. This is
// usually caused by a break in the connection.
var ErrAborted = errors.New("imap: command aborted")

// abort is a sentinel value assigned to Command.result to indicate the absence
// of a valid command completion response.
var abort = new(Response)

// Command represents a single command sent to the server.
type Command struct {
	// FIFO queue for command data. These are the responses that were accepted
	// by this command's filter. New responses are appended to the end as they
	// are received.
	Data []*Response

	// Client that created this Command instance.
	client *Client

	// Command execution parameters copied from Client.CommandConfig.
	config CommandConfig

	// Command tag assigned by the Client.
	tag string

	// UID flag for FETCH, STORE, COPY, and SEARCH commands.
	uid bool

	// Command name without the UID prefix.
	name string

	// Sequence numbers or UIDs of messages affected by this command. This is
	// used to filter FETCH responses.
	seqset *SeqSet

	// Raw command text without CRLFs or literal strings.
	raw string

	// Command completion response. This is set to abort if the command is not
	// in progress, but a valid completion response was not received.
	result *Response
}

// newCommand initializes and returns a new Command instance. Nil is returned if
// the specified name does not appear in c.CommandConfig.
func newCommand(c *Client, name string) *Command {
	config := c.CommandConfig[name]
	if config == nil {
		return nil
	}
	cmd := &Command{client: c, config: *config, name: name, raw: name}
	if len(name) > 4 && name[:4] == "UID " {
		cmd.uid = true
		cmd.name = name[4:]
	}
	return cmd
}

// Client returns the Client instance that created this command.
func (cmd *Command) Client() *Client {
	return cmd.client
}

// Tag returns the command tag assigned by the Client.
func (cmd *Command) Tag() string {
	return cmd.tag
}

// UID returns true if the command is using UIDs instead of message sequence
// numbers.
func (cmd *Command) UID() bool {
	return cmd.uid
}

// Name returns the command name. If full == true, the UID prefix is included
// for UID commands.
func (cmd *Command) Name(full bool) string {
	if full && cmd.uid {
		return "UID " + cmd.name
	}
	return cmd.name
}

// InProgress returns true until the command completion result is available. No
// new responses will be appended to cmd.Data after this method returns false.
func (cmd *Command) InProgress() bool {
	return cmd.result == nil
}

// Result returns the command completion result. The call blocks until the
// command is no longer in progress. If expect != 0, an error is returned if the
// completion status is other than expected. ErrAborted is returned if the
// command execution was interrupted prior to receiving a completion response.
func (cmd *Command) Result(expect RespStatus) (rsp *Response, err error) {
	for cmd.result == nil {
		if err = cmd.client.Recv(block); err != nil {
			return
		}
	}
	if rsp = cmd.result; rsp == abort {
		rsp, err = nil, ErrAborted
	} else if expect != 0 && rsp.Status&expect == 0 {
		err = ResponseError{rsp, "unexpected completion status"}
	}
	return
}

// String returns the raw command text without CRLFs or literal data.
func (cmd *Command) String() string {
	return cmd.raw
}

// rawCommand contains the raw text and literals about to be sent to the server.
type rawCommand struct {
	*bytes.Buffer // Command text, including all required CRLFs

	literals []Literal // Literal strings
	nonsync  bool      // Support for non-synchronizing literals (RFC 2088)
	binary   bool      // Support for binary literals (RFC 3516)
}

// build returns a rawCommand struct constructed from the command parameters.
func (cmd *Command) build(tag string, fields []Field) (*rawCommand, error) {
	raw := &rawCommand{
		Buffer:  bytes.NewBuffer(make([]byte, 0, 128)),
		nonsync: cmd.client.Caps["LITERAL+"],
		binary:  cmd.client.Caps["BINARY"],
	}
	raw.WriteString(tag)
	raw.WriteByte(' ')
	if cmd.uid {
		raw.WriteString("UID ")
	}
	raw.WriteString(cmd.name)
	err := raw.WriteFields(fields, true)
	buf := raw.Bytes()
	raw.Write(crlf)

	if len(fields) > 0 {
		cmd.seqset, _ = fields[0].(*SeqSet)
	}
	if len(raw.literals) > 0 {
		buf = bytes.Replace(buf, crlf, nil, -1)
	}
	cmd.tag = tag
	cmd.raw = string(buf)
	return raw, err
}

// WriteFields writes command fields to the raw buffer using the appropriate
// format for each field type.
func (raw *rawCommand) WriteFields(fields []Field, SP bool) error {
	for _, f := range fields {
		if SP {
			raw.WriteByte(' ')
		} else {
			SP = true
		}
		switch v := f.(type) {
		case string:
			raw.WriteString(v)
		case int, int8, int16, int32, int64:
			raw.WriteString(strconv.FormatInt(intValue(f), 10))
		case uint, uint8, uint16, uint32, uint64:
			raw.WriteString(strconv.FormatUint(uintValue(f), 10))
		case time.Time:
			raw.WriteString(v.Format(DATETIME))
		case []Field:
			raw.WriteByte('(')
			if err := raw.WriteFields(v, false); err != nil {
				return err
			}
			raw.WriteByte(')')
		case []byte:
			raw.Write(v)
		case Literal:
			info := v.Info()
			if info.Bin {
				if !raw.binary {
					return NotAvailableError("BINARY")
				}
				raw.WriteByte('~')
			}
			raw.WriteByte('{')
			raw.WriteString(strconv.FormatUint(uint64(info.Len), 10))
			if raw.nonsync {
				raw.WriteByte('+')
			}
			raw.WriteString("}\r\n")
			raw.literals = append(raw.literals, v)
		case fmt.Stringer:
			raw.WriteString(v.String())
		case nil:
			raw.WriteString("NIL")
		default:
			return fmt.Errorf("imap: invalid command field %#v", v)
		}
	}
	return nil
}

// ReadLine returns the next line from the raw buffer, panicking if a complete
// line is not found. The CRLF ending is stripped. The line remains valid until
// the next read or write call.
func (raw *rawCommand) ReadLine() []byte {
	b := raw.Bytes()
	n := bytes.IndexByte(b, '\n') + 1
	if n < 2 || b[n-2] != '\r' {
		panic("imap: corrupt command text buffer") // Should never happen...
	}
	return raw.Next(n)[:n-2]
}

// ResponseFilter defines the signature of functions that determine response
// ownership. The function returns true if rsp belongs to cmd. A nil filter
// rejects all responses. A response that is rejected by all active filters is
// considered to be unilateral server data.
type ResponseFilter func(cmd *Command, rsp *Response) bool

// NameFilter accepts the response if rsp.Label matches the command name.
func NameFilter(cmd *Command, rsp *Response) bool {
	return rsp.Label == cmd.name
}

// ByeFilter accepts the response if rsp.Status is BYE.
func ByeFilter(_ *Command, rsp *Response) bool {
	return rsp.Status == BYE
}

// FetchFilter accepts FETCH and STORE command responses by matching message
// sequence numbers or UIDs, depending on the command type. UID matches are more
// exact because there is no risk of mistaking unilateral server data (e.g. an
// unsolicited flags update) for command data.
func FetchFilter(cmd *Command, rsp *Response) bool {
	msg := rsp.MessageInfo()
	if msg == nil {
		return false // Not a FETCH response
	} else if cmd.seqset == nil {
		return true // Accept all FETCH responses if SeqSet wasn't used
	}
	set := *cmd.seqset

	// Check message sequence number or UID against the set
	if cmd.uid {
		if msg.UID == 0 {
			return false // UID data item must be included for UID commands
		} else if set.Contains(msg.UID) {
			return true
		}
	} else if set.Contains(msg.Seq) {
		return true
	}

	// Try matching against "*"
	return set.Dynamic() && msg.Seq == cmd.client.Mailbox.Messages
}

// LabelFilter returns a new filter configured to accept responses with the
// specified labels.
func LabelFilter(labels ...string) ResponseFilter {
	accept := make(map[string]bool, len(labels))
	for _, v := range labels {
		accept[v] = true
	}
	return func(_ *Command, rsp *Response) bool {
		return accept[rsp.Label]
	}
}

// SelectFilter accepts SELECT and EXAMINE command responses.
var SelectFilter = LabelFilter(
	"FLAGS", "EXISTS", "RECENT",
	"UNSEEN", "PERMANENTFLAGS", "UIDNEXT", "UIDVALIDITY",
	"UIDNOTSTICKY",
)

// CommandConfig specifies command execution parameters.
type CommandConfig struct {
	States    ConnState      // Mask of states in which this command may be issued
	Filter    ResponseFilter // Filter for identifying command responses
	Exclusive bool           // Exclusive Client access flag
}

// defaultCommands returns the default command configuration map used to
// initialize Client.CommandConfig.
func defaultCommands() map[string]*CommandConfig {
	const (
		all   = Login | Auth | Selected | Logout
		login = Login
		auth  = Auth | Selected
		sel   = Selected
	)
	return map[string]*CommandConfig{
		// RFC 3501 (6.1. Client Commands - Any State)
		"CAPABILITY": &CommandConfig{States: all, Filter: NameFilter},
		"NOOP":       &CommandConfig{States: all},
		"LOGOUT":     &CommandConfig{States: all, Filter: ByeFilter},

		// RFC 3501 (6.2. Client Commands - Not Authenticated State)
		"STARTTLS":     &CommandConfig{States: login, Exclusive: true},
		"AUTHENTICATE": &CommandConfig{States: login, Exclusive: true},
		"LOGIN":        &CommandConfig{States: login, Exclusive: true},

		// RFC 3501 (6.3. Client Commands - Authenticated State)
		"SELECT":      &CommandConfig{States: auth, Filter: SelectFilter, Exclusive: true},
		"EXAMINE":     &CommandConfig{States: auth, Filter: SelectFilter, Exclusive: true},
		"CREATE":      &CommandConfig{States: auth},
		"DELETE":      &CommandConfig{States: auth},
		"RENAME":      &CommandConfig{States: auth},
		"SUBSCRIBE":   &CommandConfig{States: auth},
		"UNSUBSCRIBE": &CommandConfig{States: auth},
		"LIST":        &CommandConfig{States: auth, Filter: NameFilter},
		"LSUB":        &CommandConfig{States: auth, Filter: NameFilter},
		"STATUS":      &CommandConfig{States: auth, Filter: NameFilter},
		"APPEND":      &CommandConfig{States: auth},

		// RFC 3501 (6.4. Client Commands - Selected State)
		"CHECK":      &CommandConfig{States: sel},
		"CLOSE":      &CommandConfig{States: sel, Exclusive: true},
		"EXPUNGE":    &CommandConfig{States: sel, Filter: NameFilter},
		"SEARCH":     &CommandConfig{States: sel, Filter: NameFilter},
		"FETCH":      &CommandConfig{States: sel, Filter: FetchFilter},
		"STORE":      &CommandConfig{States: sel, Filter: FetchFilter},
		"COPY":       &CommandConfig{States: sel},
		"UID SEARCH": &CommandConfig{States: sel, Filter: NameFilter},
		"UID FETCH":  &CommandConfig{States: sel, Filter: FetchFilter},
		"UID STORE":  &CommandConfig{States: sel, Filter: FetchFilter},
		"UID COPY":   &CommandConfig{States: sel},

		// RFC 6851
		"MOVE":     &CommandConfig{States: sel},
		"UID MOVE": &CommandConfig{States: sel},

		// RFC 2087
		"SETQUOTA":     &CommandConfig{States: auth, Filter: LabelFilter("QUOTA")},
		"GETQUOTA":     &CommandConfig{States: auth, Filter: LabelFilter("QUOTA")},
		"GETQUOTAROOT": &CommandConfig{States: auth, Filter: LabelFilter("QUOTA", "QUOTAROOT")},

		// RFC 2177
		"IDLE": &CommandConfig{States: auth, Exclusive: true},

		// RFC 2971
		"ID": &CommandConfig{States: all, Filter: NameFilter},

		// RFC 3691
		"UNSELECT": &CommandConfig{States: sel, Exclusive: true},

		// RFC 4315
		"UID EXPUNGE": &CommandConfig{States: sel, Filter: NameFilter},

		// RFC 4978
		"COMPRESS": &CommandConfig{States: auth, Exclusive: true},

		// RFC 5161
		"ENABLE": &CommandConfig{States: all, Filter: LabelFilter("ENABLED")},
	}
}
