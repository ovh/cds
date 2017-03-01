// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// ParserError indicates a problem with the server response format. This could
// be the result of an unsupported extension or nonstandard server behavior.
type ParserError struct {
	Info   string // Short message explaining the problem
	Line   []byte // Full or partial response line, starting with the tag
	Offset int    // Parser offset, starting at 0
}

func (err *ParserError) Error() string {
	if err.Line == nil {
		return "imap: " + err.Info
	}
	line, ellipsis := err.Line, ""
	if len(line) > rawLimit {
		line, ellipsis = line[:rawLimit], "..."
	}
	return fmt.Sprintf("imap: %s at offset %d of %+q%s",
		err.Info, err.Offset, line, ellipsis)
}

// readerInput is the interface for reading all parts of a response. This
// interface is implemented by transport.
type readerInput interface {
	io.Reader
	ReadLine() (line []byte, err error)
}

// reader creates rawResponse structs and provides additional lines and literals
// to the parser when requested.
type reader struct {
	readerInput
	LiteralReader

	tagid []byte // Tag prefix expected in command completion responses ([A-Z]+)
	order int64  // Response order counter
}

// rawResponse is an intermediate response form used to construct full Response
// objects. The struct returned by reader.Next() contains the start of the next
// response up to the first literal string, if there is one. The parser reads
// literals and additional lines as needed via reader.More(), which appends new
// bytes to line and tail. The current parser position can be calculated as
// len(raw.line) - len(raw.tail).
type rawResponse struct {
	*Response
	*reader

	line []byte // Full response line without literals or CRLFs
	tail []byte // Unconsumed line ending (parser state)
}

// newReader returns a reader configured to accept tagged responses beginning
// with tagid.
func newReader(in readerInput, lr LiteralReader, tagid string) *reader {
	if in == nil || lr == nil || len(tagid) == 0 {
		panic("imap: bad arguments to newReader")
	}
	for _, c := range tagid {
		if c < 'A' || c > 'Z' {
			panic("imap: bad tagid format")
		}
	}
	return &reader{in, lr, []byte(tagid), 0}
}

// Next returns the next unparsed server response, or any data read prior to an
// error. If an error is returned and rsp != nil, the connection should be
// terminated because the client and server are no longer synchronized.
func (r *reader) Next() (raw *rawResponse, err error) {
	raw = &rawResponse{reader: r}
	if raw.line, err = r.ReadLine(); err != nil {
		if len(raw.line) == 0 {
			raw = nil
		}
	} else if tag := r.tag(raw.line); tag != "" {
		r.order++
		raw.Response = &Response{Order: r.order, Raw: raw.line, Tag: tag}
		raw.tail = raw.line[len(tag)+1:]
	} else {
		err = &ProtocolError{"bad response tag", raw.line}
	}
	return
}

// More returns the next literal string and reads one more line from the server.
func (r *reader) More(raw *rawResponse, i LiteralInfo) (l Literal, err error) {
	src := io.LimitedReader{R: r, N: int64(i.Len)}
	if l, err = r.ReadLiteral(&src, i); l != nil {
		raw.Literals = append(raw.Literals, l)
		if err == nil {
			var line []byte
			if line, err = r.ReadLine(); len(line) > 0 { // ok if err != nil
				pos := raw.pos()
				raw.line = append(raw.line, line...)
				raw.tail = raw.line[pos:]
				raw.Raw = raw.line
			}
		}
	} else if err == nil {
		// Sanity check for user-provided ReadLiteral implementations
		panic("imap: ReadLiteral returned (nil, nil)")
	}
	return
}

// tag verifies that line is a valid start of a new server response and returns
// the full response tag. Valid tags are "*" (untagged status/data), "+"
// (continuation request), and strings in the format "{r.tagid}[0-9]+" (command
// completion). The tag must be followed by a space.
func (r *reader) tag(line []byte) string {
	if n := bytes.IndexByte(line, ' '); n == 1 {
		if c := line[0]; c == '*' || c == '+' {
			return string(c)
		}
	} else if i := len(r.tagid); i < n && bytes.Equal(line[:i], r.tagid) {
		for _, c := range line[i:n] {
			if c < '0' || c > '9' {
				return ""
			}
		}
		return string(line[:n])
	}
	return ""
}

// Error returned by parseCondition to indicate that rsp.Type != Status.
var errNotStatus error = &ParserError{Info: "not a status response"}

// Parse converts rawResponse into a full Response object by calling parseX
// methods, which gradually consume raw.tail.
func (raw *rawResponse) Parse() (rsp *Response, err error) {
	if raw.Response == nil {
		return nil, &ParserError{"unparsable response", raw.line, 0}
	}
	switch rsp = raw.Response; rsp.Tag {
	case "*":
		if err = raw.parseCondition(OK | NO | BAD | PREAUTH | BYE); err == nil {
			rsp.Type = Status
			err = raw.parseStatus()
		} else if err == errNotStatus {
			rsp.Type = Data
			rsp.Fields, err = raw.parseFields(nul)
			if len(rsp.Fields) == 0 && err == nil {
				err = raw.error("empty data response", 0)
			}
		}
	case "+":
		rsp.Type = Continue
		raw.parseContinue()
	default:
		if err = raw.parseCondition(OK | NO | BAD); err == nil {
			rsp.Type = Done
			err = raw.parseStatus()
		} else if err == errNotStatus {
			err = &ParserError{"unknown response type", raw.line, 0}
		}
	}
	if len(raw.tail) > 0 && err == nil {
		err = raw.unexpected(0)
	}
	raw.Response = nil
	return
}

// pos returns the current parser position in raw.line.
func (raw *rawResponse) pos() int {
	return len(raw.line) - len(raw.tail)
}

// error returns a ParserError to indicate a problem with the response at the
// specified offset. The offset is relative to raw.tail.
func (raw *rawResponse) error(info string, off int) error {
	return &ParserError{info, raw.line, raw.pos() + off}
}

// unexpected returns a ParserError to indicate an unexpected byte at the
// specified offset. The offset is relative to raw.tail.
func (raw *rawResponse) unexpected(off int) error {
	c := raw.line[raw.pos()+off]
	return raw.error(fmt.Sprintf("unexpected %+q", c), off)
}

// missing returns a ParserError to indicate the absence of a required character
// or section at the specified offset. The offset is relative to raw.tail.
func (raw *rawResponse) missing(v interface{}, off int) error {
	if _, ok := v.(byte); ok {
		return raw.error(fmt.Sprintf("missing %+q", v), off)
	}
	return raw.error(fmt.Sprintf("missing %v", v), off)
}

// Valid status conditions.
var bStatus = []struct {
	b []byte
	s RespStatus
}{
	{[]byte("OK"), OK},
	{[]byte("NO"), NO},
	{[]byte("BAD"), BAD},
	{[]byte("PREAUTH"), PREAUTH},
	{[]byte("BYE"), BYE},
}

// parseCondition extracts the status condition if raw is a status response
// (ABNF: resp-cond-*). errNotStatus is returned for all other response types.
func (raw *rawResponse) parseCondition(accept RespStatus) error {
outer:
	for _, v := range bStatus {
		if n := len(v.b); n <= len(raw.tail) {
			for i, c := range v.b {
				if raw.tail[i]&0xDF != c { // &0xDF converts [a-z] to upper case
					continue outer
				}
			}
			if n == len(raw.tail) {
				return raw.missing("SP", n)
			} else if raw.tail[n] == ' ' {
				if accept&v.s == 0 {
					return raw.error("unacceptable status condition", 0)
				}
				raw.Status = v.s
				raw.tail = raw.tail[n+1:]
				return nil
			}
			// Assume data response with a matching prefix (e.g. "* NOT STATUS")
			break
		}
	}
	return errNotStatus
}

// parseStatus extracts the optional response code and required text after the
// status condition (ABNF: resp-text).
func (raw *rawResponse) parseStatus() error {
	if len(raw.tail) > 0 && raw.tail[0] == '[' {
		var err error
		raw.tail = raw.tail[1:]
		if raw.Fields, err = raw.parseFields(']'); err != nil {
			return err
		} else if len(raw.Fields) == 0 {
			return raw.error("empty response code", -1)
		} else if len(raw.tail) == 0 {
			// Some servers do not send any text after the response code
			// (e.g. "* OK [UNSEEN 1]"). This is not allowed, according to RFC
			// 3501 ABNF, but we accept it for compatibility with other clients.
			raw.tail = nil
			return nil
		} else if raw.tail[0] != ' ' {
			return raw.missing("SP", 0)
		}
		raw.tail = raw.tail[1:]
	}
	if len(raw.tail) == 0 {
		return raw.missing("status text", 0)
	}
	raw.Info = string(raw.tail)
	raw.tail = nil
	return nil
}

// parseContinue extracts the text or Base64 data from a continuation request
// (ABNF: continue-req). Base64 data is saved in its original form to raw.Info,
// and decoded as []byte into raw.Fields[0].
func (raw *rawResponse) parseContinue() {
	if n := len(raw.tail); n == 0 {
		raw.Label = "BASE64"
		raw.Fields = []Field{[]byte(nil)}
	} else if n&3 == 0 {
		if b, err := b64dec(raw.tail); err == nil {
			raw.Label = "BASE64"
			raw.Fields = []Field{b}
		}
	}
	// ABNF uses resp-text, but section 7.5 states "The remainder of this
	// response is a line of text." Assume that response codes are not allowed.
	raw.Info = string(raw.tail)
	raw.tail = nil
}

// parseFields extracts as many data fields from raw.tail as possible until it
// finds the stop byte in a delimiter position. An error is returned if the stop
// byte is not found. NUL stop causes all of raw.tail to be consumed (NUL does
// not appear anywhere in raw.line - checked by transport).
func (raw *rawResponse) parseFields(stop byte) (fields []Field, err error) {
	if len(raw.tail) > 0 && raw.tail[0] == stop {
		// Empty parenthesized list, BODY[] and friends, or an error
		raw.tail = raw.tail[1:]
		return
	}
	for len(raw.tail) > 0 && err == nil {
		var f Field
		switch raw.next() {
		case QuotedString:
			f, err = raw.parseQuotedString()
		case LiteralString:
			f, err = raw.parseLiteralString()
		case List:
			raw.tail = raw.tail[1:]
			f, err = raw.parseFields(')')
		default:
			f, err = raw.parseAtom(raw.Type == Data && stop != ']')
		}
		if err == nil || f != nil {
			fields = append(fields, f)
		}
		// Delimiter
		if len(raw.tail) > 0 && err == nil {
			switch raw.tail[0] {
			case ' ':
				// Allow a space even if it's at the end of the response. Yahoo
				// servers send "* SEARCH 2 84 882 " in violation of RFC 3501.
				raw.tail = raw.tail[1:]
			case stop:
				raw.tail = raw.tail[1:]
				return
			case '(':
				// body-type-mpart is 1*body without a space in between
				if len(raw.tail) == 1 {
					err = raw.unexpected(0)
				}
			default:
				err = raw.unexpected(0)
			}
		}
	}
	if stop != nul && err == nil {
		err = raw.missing(stop, 0)
	}
	return
}

// next returns the type of the next response field. The default type is Atom,
// which includes atoms, numbers, and NILs.
func (raw *rawResponse) next() FieldType {
	switch raw.tail[0] {
	case '"':
		return QuotedString
	case '{':
		return LiteralString
	case '(':
		return List

	// RFC 5738 utf8-quoted
	case '*':
		if len(raw.tail) >= 2 && raw.tail[1] == '"' {
			return QuotedString
		}

	// RFC 3516 literal8
	case '~':
		if len(raw.tail) >= 2 && raw.tail[1] == '{' {
			return LiteralString
		}
	}
	return Atom
}

// parseQuotedString returns the next quoted string. The string stays quoted,
// but validation is performed to ensure that subsequent calls to Unquote() are
// successful.
func (raw *rawResponse) parseQuotedString() (f Field, err error) {
	start := 1
	if raw.tail[0] == '*' {
		start++
	}
	escaped := false
	for n, c := range raw.tail[start:] {
		if escaped {
			escaped = false
		} else if c == '\\' {
			escaped = true
		} else if c == '"' {
			n += start + 1
			if _, ok := UnquoteBytes(raw.tail[:n]); ok {
				f = string(raw.tail[:n])
				raw.tail = raw.tail[n:]
				return
			}
			break
		}
	}
	err = raw.error("bad quoted string", 0)
	return
}

// parseLiteralString returns the next literal string. The octet count should be
// the last field in raw.tail. An additional line of text will be appended to
// raw.line and raw.tail after the literal is received.
func (raw *rawResponse) parseLiteralString() (f Field, err error) {
	var info LiteralInfo
	start := 1
	if raw.tail[0] == '~' {
		info.Bin = true
		start++
	}
	n := len(raw.tail) - 1
	if n-start < 1 || raw.tail[n] != '}' {
		err = raw.unexpected(0)
		return
	}
	oc, err := strconv.ParseUint(string(raw.tail[start:n]), 10, 32)
	if err != nil {
		err = raw.error("bad literal octet count", start)
		return
	}
	info.Len = uint32(oc)
	if f, err = raw.More(raw, info); err == nil {
		raw.tail = raw.tail[n+1:]
	}
	return
}

// atomSpecials identifies ASCII characters that either may not appear in atoms
// or require special handling (ABNF: ATOM-CHAR).
var atomSpecials [char]bool

func init() {
	// atom-specials + '[' to provide special handling for BODY[...]
	s := []byte{'(', ')', '{', ' ', '%', '*', '"', '[', '\\', ']', '\x7F'}
	for c := byte(0); c < char; c++ {
		atomSpecials[c] = c < ctl || bytes.IndexByte(s, c) >= 0
	}
}

// parseAtom returns the next atom, number, or NIL. The syntax rules are relaxed
// to treat sequences such as "BODY[...]<...>" as a single atom. Numbers are
// converted to uint32, NIL is converted to nil, everything else becomes a
// string. Flags (e.g. "\Seen") are converted to title case, other strings are
// left in their original form.
func (raw *rawResponse) parseAtom(astring bool) (f Field, err error) {
	n, flag := 0, false
	for end := len(raw.tail); n < end; n++ {
		if c := raw.tail[n]; c >= char || atomSpecials[c] {
			switch c {
			case '\\':
				if n == 0 {
					flag = true
					astring = false
					continue // ABNF: flag (e.g. `\Seen`)
				}
			case '*':
				if n == 1 && flag {
					n++ // ABNF: flag-perm (`\*`), end of atom
				}
			case '[':
				if n == 4 && bytes.EqualFold(raw.tail[:4], []byte("BODY")) {
					pos := raw.pos()
					raw.tail = raw.tail[n+1:] // Temporary shift for parseFields

					// TODO: Literals between '[' and ']' are handled correctly,
					// but only the octet count will make it into the returned
					// atom. Would any server actually send a literal here, and
					// is it a problem to discard it since the client already
					// knows what was requested?
					if _, err = raw.parseFields(']'); err != nil {
						return
					}
					n = raw.pos() - pos - 1
					raw.tail = raw.line[pos:] // Undo temporary shift
					end = len(raw.tail)
					astring = false
				}
				continue // ABNF: fetch-att ("BODY[...]<...>"), atom, or astring
			case ']':
				if astring {
					continue // ABNF: ASTRING-CHAR
				}
			}
			break // raw.tail[n] is a delimiter or an unexpected byte
		}
	}

	// Atom must have at least one character, two if it starts with a backslash
	if n < 2 && (n == 0 || flag) {
		err = raw.unexpected(0)
		return
	}

	// Take whatever was found, let parseFields report delimiter errors
	atom := raw.tail[:n]
	if norm := normalize(atom); flag {
		f = norm
	} else if norm != "NIL" {
		if c := norm[0]; '0' <= c && c <= '9' {
			if ui, err := strconv.ParseUint(norm, 10, 32); err == nil {
				f = uint32(ui)
			}
		}
		if f == nil {
			if raw.Label == "" {
				raw.Label = norm
			}
			f = string(atom)
		}
	}
	raw.tail = raw.tail[n:]
	return
}

// normalize returns a normalized string copy of an atom. Non-flag atoms are
// converted to upper case. Flags are converted to title case (e.g. `\Seen`).
func normalize(atom []byte) string {
	norm := []byte(nil)
	want := byte(0) // Want upper case
	for i, c := range atom {
		have := c & 0x20
		if c &= 0xDF; 'A' <= c && c <= 'Z' && have != want {
			norm = make([]byte, len(atom))
			break
		} else if i == 1 && atom[0] == '\\' {
			want = 0x20 // Want lower case starting at i == 2
		}
	}
	if norm == nil {
		return string(atom) // Fast path: no changes
	}
	want = 0
	for i, c := range atom {
		if c &= 0xDF; 'A' <= c && c <= 'Z' {
			norm[i] = c | want
		} else {
			norm[i] = atom[i]
		}
		if i == 1 && atom[0] == '\\' {
			want = 0x20
		}
	}
	return string(norm)
}
