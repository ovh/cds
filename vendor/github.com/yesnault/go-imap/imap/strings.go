// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"io"
	"unicode/utf8"
)

const (
	nul  = 0x00
	ctl  = 0x20
	char = 0x80
	cr   = '\r'
	lf   = '\n'
)

// Quote returns the input as a quoted string for use in a command. An empty
// string is returned if the input cannot be quoted and must be sent as a
// literal. Setting utf8quoted to true indicates server support for utf8-quoted
// string format, as described in RFC 5738. The utf8-quoted form will be used
// only if the input contains non-ASCII characters.
func Quote(s string, utf8quoted bool) string {
	return string(QuoteBytes([]byte(s), utf8quoted))
}

// QuoteBytes returns the input as a quoted byte slice. Nil is returned if the
// input cannot be quoted and must be sent as a literal.
func QuoteBytes(s []byte, utf8quoted bool) []byte {
	escape, unicode := 0, false
	for i, n := 0, len(s); i < n; {
		if c := s[i]; c < char {
			if c == '"' || c == '\\' {
				escape++
			} else if c < ctl && (c == nul || c == cr || c == lf) {
				return nil
			}
			i++
		} else if !utf8quoted {
			return nil
		} else {
			_, size := utf8.DecodeRune(s[i:])
			if size == 1 {
				return nil
			}
			unicode = true
			i += size
		}
	}
	q := make([]byte, 0, len(s)+escape+3)
	if unicode {
		q = append(q, '*')
	}
	q = append(q, '"')
	if escape == 0 {
		q = append(q, s...)
	} else {
		for _, c := range s {
			if c == '"' || c == '\\' {
				q = append(q, '\\', c)
			} else {
				q = append(q, c)
			}
		}
	}
	return append(q, '"')
}

// Quoted returns true if a string or []byte appears to contain a quoted string,
// based on the presence of surrounding double quotes. The string contents are
// not checked, so it may still contain illegal characters or escape sequences.
// The string may be encoded in utf8-quoted format, as described in RFC 5738.
func Quoted(f Field) bool {
	switch s := f.(type) {
	case string:
		if n := len(s); n >= 2 && s[n-1] == '"' {
			return s[0] == '"' || (n >= 3 && s[0] == '*' && s[1] == '"')
		}
	case []byte:
		if n := len(s); n >= 2 && s[n-1] == '"' {
			return s[0] == '"' || (n >= 3 && s[0] == '*' && s[1] == '"')
		}
	}
	return false
}

// QuotedUTF8 returns true if a string or []byte appears to contain a quoted
// string encoded in utf8-quoted format.
func QuotedUTF8(f Field) bool {
	switch s := f.(type) {
	case string:
		n := len(s)
		return n >= 3 && s[0] == '*' && s[1] == '"' && s[n-1] == '"'
	case []byte:
		n := len(s)
		return n >= 3 && s[0] == '*' && s[1] == '"' && s[n-1] == '"'
	}
	return false
}

// Unquote is the reverse of Quote. An empty string is returned and ok is set to
// false if the input is not a valid quoted string. RFC 3501 specifications are
// relaxed to accept all valid UTF-8 encoded strings with or without the use of
// utf8-quoted format (RFC 5738). Rules disallowing the use of NUL, CR, and LF
// characters still apply. All (and only) double quote and backslash characters
// must be escaped with a backslash.
func Unquote(q string) (s string, ok bool) {
	if Quoted(q) {
		var b []byte
		if b, ok = unquote([]byte(q)); len(b) > 0 {
			s = string(b)
		}
	}
	return
}

// UnquoteBytes is the reverse of QuoteBytes.
func UnquoteBytes(q []byte) (s []byte, ok bool) {
	if Quoted(q) {
		s, ok = unquote(q)
	}
	return
}

// unquote performs the actual unquote operation on a byte slice. It assumes
// that Quoted(q) == true.
func unquote(q []byte) (s []byte, ok bool) {
	n := len(q)
	if q[0] == '"' {
		q = q[1 : n-1] // "..."
	} else {
		q = q[2 : n-1] // *"..."
	}
	if n = len(q); n == 0 {
		ok = true
		return
	}
	b := make([]byte, 0, n)
	for i := 0; i < n; {
		if c := q[i]; c < char {
			if c == '\\' {
				if i++; i == n {
					return
				} else if c = q[i]; c != '"' && c != '\\' {
					return
				}
			} else if c < ctl && (c == nul || c == cr || c == lf) || c == '"' {
				return
			}
			b = append(b, c)
			i++
		} else {
			_, size := utf8.DecodeRune(q[i:])
			if size == 1 {
				return
			}
			b = append(b, q[i:i+size]...)
			i += size
		}
	}
	return b, true
}

// LiteralInfo describes the attributes of an incoming or outgoing literal
// string.
type LiteralInfo struct {
	Len uint32 // Literal octet count
	Bin bool   // RFC 3516 literal8 binary format flag
}

// Literal represents a single incoming or outgoing literal string, as described
// in RFC 3501 section 4.3. Incoming literals are constructed by a
// LiteralReader. The default implementation saves all literals to memory. A
// custom LiteralReader implementation can save literals directly to files. This
// could be advantageous when the client is receiving message bodies containing
// attachments several MB in size. Likewise, a custom Literal implementation can
// transmit outgoing literals by reading directly from files or other data
// sources.
type Literal interface {
	// WriteTo writes Info().Length bytes to the Writer w. For the default
	// Literal implementation, use AsString or AsBytes field functions to access
	// the incoming data directly without copying everything through a Writer.
	io.WriterTo

	// Info returns information about the contained literal.
	Info() LiteralInfo
}

// NewLiteral creates a new literal string from a byte slice. The Literal will
// point to the same underlying array as the original slice, so it is not safe
// to modify the array data until the literal has been sent in a command. It is
// the caller's responsibility to create a copy of the data, if needed.
func NewLiteral(b []byte) Literal {
	return &literal{b, LiteralInfo{Len: uint32(len(b))}}
}

// NewLiteral8 creates a new binary literal string from a byte slice. This
// literal is sent using the literal8 syntax, as described in RFC 3516. The
// server must advertise "BINARY" capability for such literals to be accepted.
func NewLiteral8(b []byte) Literal {
	return &literal{b, LiteralInfo{Len: uint32(len(b)), Bin: true}}
}

// literal stores a single literal string in a byte slice.
type literal struct {
	data []byte
	info LiteralInfo
}

func (l *literal) WriteTo(w io.Writer) (n int64, err error) {
	if len(l.data) == 0 {
		return
	}
	nn, err := w.Write(l.data)
	n = int64(nn)
	return
}

func (l *literal) Info() LiteralInfo {
	return l.info
}

// LiteralReader is the interface for receiving literal strings from the server.
//
// ReadLiteral reads exactly i.Length bytes from r into a new literal. It must
// return a Literal instance even when i.Length == 0 (empty string). A return
// value of (nil, nil) is invalid.
type LiteralReader interface {
	ReadLiteral(r io.Reader, i LiteralInfo) (Literal, error)
}

// MemoryReader implements the LiteralReader interface by saving all incoming
// literals to memory.
type MemoryReader struct{}

func (MemoryReader) ReadLiteral(r io.Reader, i LiteralInfo) (Literal, error) {
	if i.Len == 0 {
		return &literal{info: i}, nil
	}
	b := make([]byte, i.Len)
	n, err := io.ReadFull(r, b)
	return &literal{b[:n], i}, err
}

// toUpper returns a copy of s with all ASCII characters converted to upper
// case. This is a faster version of strings.ToUpper for ASCII-only strings.
func toUpper(s string) string {
	n := len(s)
	for i := 0; i < n; i++ {
		if c := s[i]; 'a' <= c && c <= 'z' {
			goto convert
		}
	}
	return s

convert:
	u := make([]byte, n)
	for i := 0; i < n; i++ {
		c := s[i]
		if 'a' <= c && c <= 'z' {
			c &= 0xDF
		}
		u[i] = c
	}
	return string(u)
}
