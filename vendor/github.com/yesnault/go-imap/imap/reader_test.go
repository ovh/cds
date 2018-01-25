// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"reflect"
	"testing"
)

const CRLF = "\r\n"

func TestReaderNext(t *testing.T) {
	setTag := new(Response)
	tests := []struct {
		in  string
		out *Response
	}{
		// Tag ID = "A"
		{"A", setTag},

		{"", nil},
		{" ", nil},
		{"?", nil},
		{"+", nil},
		{"*", nil},
		{" *", nil},
		{"  ", nil},
		{"! ", nil},
		{"\r ", nil},
		{"*\t", nil},
		{" * ", nil},
		{"*\x00", nil},
		{"* \x00", nil},

		{"A", nil},
		{"A ", nil},
		{"1 ", nil},
		{"A1", nil},
		{"a1 ", nil},
		{"B1 ", nil},
		{"AB ", nil},
		{"12 ", nil},
		{"AB1 ", nil},
		{"A1B ", nil},
		{"A-1 ", nil},
		{"A123", nil},

		{"* ", &Response{Raw: []byte("* "), Tag: "*"}},
		{"+ ", &Response{Raw: []byte("+ "), Tag: "+"}},
		{"A0 ", &Response{Raw: []byte("A0 "), Tag: "A0"}},
		{"A1 ", &Response{Raw: []byte("A1 "), Tag: "A1"}},
		{"A01 ", &Response{Raw: []byte("A01 "), Tag: "A01"}},
		{"A42  ", &Response{Raw: []byte("A42  "), Tag: "A42"}},
		{"A123 ", &Response{Raw: []byte("A123 "), Tag: "A123"}},
		{"A4294967295 ", &Response{Raw: []byte("A4294967295 "), Tag: "A4294967295"}},
		{"A4294967296 ", &Response{Raw: []byte("A4294967296 "), Tag: "A4294967296"}},

		// Tag ID = "AB"
		{"AB", setTag},

		{"A1 ", nil},
		{"AB ", nil},
		{"Ab1 ", nil},
		{"ABC1 ", nil},
		{"AB1 ", &Response{Raw: []byte("AB1 "), Tag: "AB1"}},

		// Tag ID = "ABC"
		{"ABC", setTag},

		{"AB1 ", nil},
		{"ABC1 ", &Response{Raw: []byte("ABC1 "), Tag: "ABC1"}},
		{"ABC123 ", &Response{Raw: []byte("ABC123 "), Tag: "ABC123"}},
	}
	c, s := newTestConn(1024)
	C := newTransport(c, nil)
	r := newReader(C, MemoryReader{}, "TAG")

	raw, err := r.Next()
	if raw != nil || err == nil {
		t.Fatalf("Next() expected timeout; got %#v (%v)", raw, err)
	}
	for _, test := range tests {
		if test.out == setTag {
			r = newReader(C, MemoryReader{}, test.in)
			continue
		}
		C.clear()
		s.Write([]byte(test.in + CRLF))

		raw, err = r.Next()
		if raw == nil {
			t.Errorf("Next(%+q) unexpected nil response; %v", test.in, err)
		} else if out := raw.Response; test.out == nil {
			if out != nil || err == nil {
				t.Errorf("Next(%+q) expected error; got\n%#v", test.in, out)
			}
		} else if err != nil {
			t.Errorf("Next(%+q) unexpected error; %v", test.in, err)
		} else if out.Order = 0; !reflect.DeepEqual(out, test.out) {
			t.Errorf("Next(%+q) expected\n%#v; got\n%#v", test.in, test.out, out)
		}
	}
}

func lit(s string) Literal {
	if s == "" {
		return NewLiteral(nil)
	}
	return NewLiteral([]byte(s))
}

func lit8(s string) Literal {
	if s == "" {
		return NewLiteral8(nil)
	}
	return NewLiteral8([]byte(s))
}

func TestReaderMore(t *testing.T) {
	tests := []struct {
		in  string
		out *Response
	}{
		{"* {" + CRLF, nil},
		{"* { " + CRLF, nil},
		{"* ~{" + CRLF, nil},
		{"* {0" + CRLF, nil},
		{"* {0 " + CRLF, nil},
		{"*  0}" + CRLF, nil},
		{"* {}" + CRLF, nil},
		{"* ~{}" + CRLF, nil},
		{"* { }" + CRLF, nil},
		{"* {~}" + CRLF, nil},
		{"* {-1}" + CRLF, nil},
		{"* {+0}" + CRLF, nil},
		{"* {+1}" + CRLF, nil},
		{"* {1,000}" + CRLF, nil},
		{"* {1.0}" + CRLF, nil},
		{"* {1a}" + CRLF, nil},
		{"* {0x1a}" + CRLF, nil},
		{"* 123{4}" + CRLF, nil},
		{"* {x}" + CRLF, nil},
		{"* {4294967296}" + CRLF, nil},

		{"* {0}", nil},
		{"* {0} " + CRLF, nil},
		//{"* {0}" + CRLF + " ", nil},
		{"* {0}" + CRLF + "{1}", nil},
		{"* {1}" + CRLF + "{1}", nil},
		{"* {1}" + CRLF + "x{1}", nil},

		{"* 0 LITERALS",
			&Response{Raw: []byte("* 0 LITERALS")}},
		{"* NO LITERALS {1}",
			&Response{Raw: []byte("* NO LITERALS {1}")}},

		{"* {0}" + CRLF,
			&Response{Raw: []byte("* {0}"), Literals: []Literal{lit("")}}},
		{"* {1}" + CRLF + "*",
			&Response{Raw: []byte("* {1}"), Literals: []Literal{lit("*")}}},
		{"* {2}" + CRLF + "* ",
			&Response{Raw: []byte("* {2}"), Literals: []Literal{lit("* ")}}},
		{"* {10}" + CRLF + "0123456789",
			&Response{Raw: []byte("* {10}"), Literals: []Literal{lit("0123456789")}}},
		{"* {03}" + CRLF + "*+ ",
			&Response{Raw: []byte("* {03}"), Literals: []Literal{lit("*+ ")}}},
		{"* ~{16}" + CRLF + "hello, world\x00\xFF\r\n",
			&Response{Raw: []byte("* ~{16}"), Literals: []Literal{lit8("hello, world\x00\xFF\r\n")}}},

		{"* ONE LITERAL {0}" + CRLF,
			&Response{Raw: []byte("* ONE LITERAL {0}"), Literals: []Literal{lit("")}}},
		{"* ONE LITERAL {1}" + CRLF + ".",
			&Response{Raw: []byte("* ONE LITERAL {1}"), Literals: []Literal{lit(".")}}},
		{"* {3}" + CRLF + "TWO ~{8}" + CRLF + "LITERALS",
			&Response{Raw: []byte("* {3} ~{8}"), Literals: []Literal{lit("TWO"), lit8("LITERALS")}}},
		{"* ~{5}" + CRLF + "three {0}" + CRLF + " literal {7}" + CRLF + "strings here",
			&Response{Raw: []byte("* ~{5} {0} literal {7} here"), Literals: []Literal{lit8("three"), lit(""), lit("strings")}}},
	}
	c, s := newTestConn(1024)
	C := newTransport(c, nil)
	r := newReader(C, MemoryReader{}, "A")

	for _, test := range tests {
		C.clear()
		s.Write([]byte(test.in + CRLF))

		raw, err := r.Next()
		if raw == nil || err != nil {
			t.Errorf("Next(%+q) unexpected error; %v", test.in, err)
			continue
		}
		out, err := raw.Parse()
		if out != nil {
			out = &Response{Raw: out.Raw, Literals: out.Literals}
		}
		if test.out == nil {
			if err == nil {
				t.Errorf("Parse(%+q) expected error; got\n%#v", test.in, out)
			}
		} else if err != nil {
			t.Errorf("Parse(%+q) unexpected error; %v", test.in, err)
		} else if !reflect.DeepEqual(out, test.out) {
			t.Errorf("Parse(%+q) expected\n%#v; got\n%#v", test.in, test.out, out)
		}
	}
}

const header = "" +
	"Date: Wed, 17 Jul 1996 02:23:25 -0700 (PDT)" + CRLF +
	"From: Terry Gray <gray@cac.washington.edu>" + CRLF +
	"Subject: IMAP4rev1 WG mtg summary and minutes" + CRLF +
	"To: imap@cac.washington.edu" + CRLF +
	"cc: minutes@CNRI.Reston.VA.US, John Klensin <KLENSIN@MIT.EDU>" + CRLF +
	"Message-Id: <B27397-0100000@cac.washington.edu>" + CRLF +
	"MIME-Version: 1.0" + CRLF +
	"Content-Type: TEXT/PLAIN; CHARSET=US-ASCII" + CRLF + CRLF

func TestReaderParse(t *testing.T) {
	tests := []struct {
		in  string
		out *Response
	}{
		// Invalid status
		{"* NO", nil},
		{"* NO ", nil},
		{"* NO [", nil},
		{"* NO [ ", nil},
		{"* NO []", nil},
		{"* NO [] ", nil},
		{"* NO [] Bad", nil},
		{"* NO [ ] No", nil},
		{"* NO [x] ", nil},
		{"* NO [x Status", nil},
		{"* NO [x ] Status", nil},
		{"* NO [ x] Status", nil},
		{"* NO [(x] Status", nil},
		{"* NO [x)] Status", nil},
		{"* NO [(x ] Status", nil},
		{"* NO [ x)] Status", nil},
		{"* NO [(x )] Status", nil},
		{"* NO [( x)] Status", nil},

		// Invalid data
		{"* ", nil},
		{"*  ", nil},
		{"* (", nil},
		{"* ( ", nil},
		{"* (x", nil},
		{"* x)", nil},
		{"* (x ", nil},
		{"*  x)", nil},
		{"* ( )", nil},
		{"* ((x)", nil},
		{"* (x))", nil},
		{"* ( (x)", nil},
		{"* (x) )", nil},

		{`* "`, nil},
		{`* "\`, nil},
		{`* "\"`, nil},
		{`* *"`, nil},
		{`* *"\`, nil},
		{`* *"\"`, nil},
		{`* "x`, nil},
		{`* "x `, nil},
		{`* "x\"`, nil},
		{`* "x""`, nil},
		{`* "x"""`, nil},
		{`* "x" "`, nil},
		{`* "x\'"`, nil},
		{`* "\x'"`, nil},
		{`* "\\\"`, nil},

		{"* BODY[", nil},
		//{"* BODY[] ", nil},
		{"* BODY[ ]", nil},
		{"* BODY[]]", nil},
		{"* BODY[[]]", nil},

		{`* *`, nil},
		{`* \`, nil},
		{`* x\`, nil},
		{`* x*`, nil},
		{`* *x`, nil},
		{`* \x*`, nil},
		{`* \*x`, nil},
		{`* \**`, nil},
		{`* \\*`, nil},

		{`* atom(specials`, nil},
		{`* atom)specials`, nil},
		{`* atom{specials`, nil},
		{`* atom%specials`, nil},
		{`* atom*specials`, nil},
		{`* atom"specials`, nil},
		{`* atom\specials`, nil},

		{"* atom\x01specials", nil},
		{"* atom\x1Fspecials", nil},
		{"* atom\x7Fspecials", nil},
		{"* atom\x80specials", nil},
		{"* atom\xFEspecials", nil},
		{"* atom\xFFspecials", nil},

		// Invalid command completion
		{"A1 ", nil},
		{"A1 OK", nil},
		{"A1 BAD ", nil},
		{"A1  NO Error", nil},
		{"A1 X Invalid", nil},
		{"A1 BYE Go away!", nil},
		{"A1 PREAUTH Welcome!", nil},
		{"A1 DONE [OK] Nope...", nil},

		// Data formats
		{`* Atom aBc \flag \* BODY[ABC ("1" NIL)]<42> body[]`,
			&Response{Tag: "*", Type: Data, Label: "ATOM", Fields: []Field{
				"Atom", "aBc", `\Flag`, `\*`, `BODY[ABC ("1" NIL)]<42>`, "body[]"}}},
		{`* Astring [ ] [] [x] BODY[[] atom[specials atom]specials`,
			&Response{Tag: "*", Type: Data, Label: "ASTRING", Fields: []Field{
				"Astring", "[", "]", "[]", "[x]", "BODY[[]", "atom[specials", "atom]specials"}}},
		{`* Number -1 0 1 1.0 4294967295 4294967296`,
			&Response{Tag: "*", Type: Data, Label: "NUMBER", Fields: []Field{
				"Number", "-1", uint32(0), uint32(1), "1.0", ^uint32(0), "4294967296"}}},
		{`* Quoted "" *"" "\\\"" *"utf8"`,
			&Response{Tag: "*", Type: Data, Label: "QUOTED", Fields: []Field{
				"Quoted", `""`, `*""`, `"\\\""`, `*"utf8"`}}},
		{`* Literal {0}` + CRLF + ` {2}` + CRLF + `hi x ~{3}` + CRLF + "\x00\r\n",
			&Response{Tag: "*", Type: Data, Label: "LITERAL", Fields: []Field{
				"Literal", lit(""), lit("hi"), "x", lit8("\x00\r\n")}}},
		{`* List () ((()) (x ())) ((y) z)`,
			&Response{Tag: "*", Type: Data, Label: "LIST", Fields: []Field{
				"List", []Field(nil), []Field{[]Field{[]Field(nil)}, []Field{"x", []Field(nil)}}, []Field{[]Field{"y"}, "z"}}}},
		{`* NIL_ NIL nil Nil (NIL) "NIL"`,
			&Response{Tag: "*", Type: Data, Label: "NIL_", Fields: []Field{
				"NIL_", nil, nil, nil, []Field{nil}, `"NIL"`}}},

		// Basic status
		{`* OK All is well!`,
			&Response{Tag: "*", Type: Status, Status: OK, Info: "All is well!"}},
		{`* no Error!`,
			&Response{Tag: "*", Type: Status, Status: NO, Info: "Error!"}},
		{`* Bad PANIC!`,
			&Response{Tag: "*", Type: Status, Status: BAD, Info: "PANIC!"}},
		{`* PreAuth Welcome!`,
			&Response{Tag: "*", Type: Status, Status: PREAUTH, Info: "Welcome!"}},
		{`* bye  go away! `,
			&Response{Tag: "*", Type: Status, Status: BYE, Info: " go away! "}},

		// Basic data
		{`* NOT STATUS`,
			&Response{Tag: "*", Type: Data, Label: "NOT", Fields: []Field{"NOT", "STATUS"}}},
		{`* 42 abc`,
			&Response{Tag: "*", Type: Data, Label: "ABC", Fields: []Field{uint32(42), "abc"}}},
		{`* 0 nil ABC "XYZ"`,
			&Response{Tag: "*", Type: Data, Label: "ABC", Fields: []Field{uint32(0), nil, "ABC", `"XYZ"`}}},

		// Basic continue
		{`+ Go on...`,
			&Response{Tag: "+", Type: Continue, Info: "Go on..."}},
		{`+  `,
			&Response{Tag: "+", Type: Continue, Info: " "}},
		{`+ `,
			&Response{Tag: "+", Type: Continue, Info: "", Label: "BASE64", Fields: []Field{[]byte(nil)}}},
		{`+ badguess`,
			&Response{Tag: "+", Type: Continue, Info: "badguess", Label: "BASE64", Fields: []Field{[]byte("\x6D\xA7\x60\xB9\xEB\x2C")}}},
		{`+ QmFzZTY0IERhdGE=`,
			&Response{Tag: "+", Type: Continue, Info: "QmFzZTY0IERhdGE=", Label: "BASE64", Fields: []Field{[]byte("Base64 Data")}}},

		// Basic command completion
		{`A1 OK Done!`,
			&Response{Tag: "A1", Type: Done, Status: OK, Info: "Done!"}},
		{`A2 NO [read-write] Can't do it!`,
			&Response{Tag: "A2", Type: Done, Status: NO, Info: "Can't do it!", Label: "READ-WRITE", Fields: []Field{"read-write"}}},
		{`A3 BAD [BADCHARSET (UTF-8)] Try again.`,
			&Response{Tag: "A3", Type: Done, Status: BAD, Info: "Try again.", Label: "BADCHARSET", Fields: []Field{"BADCHARSET", []Field{"UTF-8"}}}},
		{`A4 bad [BADCHARSET ({5}` + CRLF + `UTF-8)] NO`,
			&Response{Tag: "A4", Type: Done, Status: BAD, Info: "NO", Label: "BADCHARSET", Fields: []Field{"BADCHARSET", []Field{lit("UTF-8")}}}},

		// Status with response code but no text (violates RFC 3501)
		{"* NO [x]",
			&Response{Tag: "*", Type: Status, Status: NO, Label: "X", Fields: []Field{"x"}}},
		{"* OK [UNSEEN 1]",
			&Response{Tag: "*", Type: Status, Status: OK, Label: "UNSEEN", Fields: []Field{"UNSEEN", uint32(1)}}},

		// Fetch data
		{`* 10000 Fetch (Flags (\seen \DELETED) UID 4827313 RFC822.SIZE 44827)`,
			&Response{Tag: "*", Type: Data, Label: "FETCH", Fields: []Field{
				uint32(10000), "Fetch", []Field{
					"Flags", []Field{`\Seen`, `\Deleted`},
					"UID", uint32(4827313),
					"RFC822.SIZE", uint32(44827)}},
			}},

		// Not a literal; "}" is not one of atom-specials
		{`+ Just some text {1}`,
			&Response{Tag: "+", Type: Continue, Info: "Just some text {1}"}},
		{`* OK Not a literal {42}`,
			&Response{Tag: "*", Type: Status, Status: OK, Info: "Not a literal {42}"}},
		{`* }`,
			&Response{Tag: "*", Type: Data, Label: "}", Fields: []Field{"}"}}},
		{`* 1}`,
			&Response{Tag: "*", Type: Data, Label: "1}", Fields: []Field{"1}"}}},
		{`* x }1}`,
			&Response{Tag: "*", Type: Data, Label: "X", Fields: []Field{"x", "}1}"}}},
		{`* LIST () "/" blurdybloop}`,
			&Response{Tag: "*", Type: Data, Label: "LIST", Fields: []Field{"LIST", []Field(nil), `"/"`, "blurdybloop}"}}},
		{`* LIST () "/" 1}`,
			&Response{Tag: "*", Type: Data, Label: "LIST", Fields: []Field{"LIST", []Field(nil), `"/"`, "1}"}}},
		{`* LIST () "{" 2}`,
			&Response{Tag: "*", Type: Data, Label: "LIST", Fields: []Field{"LIST", []Field(nil), `"{"`, "2}"}}},

		// Page 25
		{`* CAPABILITY IMAP4rev1 STARTTLS AUTH=GSSAPI`,
			&Response{Tag: "*", Type: Data, Label: "CAPABILITY", Fields: []Field{"CAPABILITY", "IMAP4rev1", "STARTTLS", "AUTH=GSSAPI"}}},
		{`* 14 FETCH (FLAGS (\Seen \Deleted))`,
			&Response{Tag: "*", Type: Data, Label: "FETCH", Fields: []Field{uint32(14), "FETCH", []Field{"FLAGS", []Field{`\Seen`, `\Deleted`}}}}},
		{`A047 OK NOOP completed`,
			&Response{Tag: "A047", Type: Done, Status: OK, Info: "NOOP completed"}},

		// Page 30
		{`+ YDMGCSqGSIb3EgECAgIBAAD/////6jcyG4GE3KkTzBeBiVHeceP2CWY0SR0fAQAgAAQEBAQ=`,
			&Response{Tag: "+", Type: Continue, Info: "YDMGCSqGSIb3EgECAgIBAAD/////6jcyG4GE3KkTzBeBiVHeceP2CWY0SR0fAQAgAAQEBAQ=", Label: "BASE64", Fields: []Field{[]byte("" +
				"\x60\x33\x06\x09\x2A\x86\x48\x86\xF7\x12\x01\x02\x02\x02\x01\x00\x00\xFF" +
				"\xFF\xFF\xFF\xEA\x37\x32\x1B\x81\x84\xDC\xA9\x13\xCC\x17\x81\x89\x51\xDE" +
				"\x71\xE3\xF6\x09\x66\x34\x49\x1D\x1F\x01\x00\x20\x00\x04\x04\x04\x04")}}},

		// Page 33
		{`* 172 EXISTS`,
			&Response{Tag: "*", Type: Data, Label: "EXISTS", Fields: []Field{uint32(172), "EXISTS"}}},
		{`* 1 RECENT`,
			&Response{Tag: "*", Type: Data, Label: "RECENT", Fields: []Field{uint32(1), "RECENT"}}},
		{`* OK [UNSEEN 12] Message 12 is first unseen`,
			&Response{Tag: "*", Type: Status, Status: OK, Info: "Message 12 is first unseen", Label: "UNSEEN", Fields: []Field{"UNSEEN", uint32(12)}}},
		{`* OK [UIDVALIDITY 3857529045] UIDs valid`,
			&Response{Tag: "*", Type: Status, Status: OK, Info: "UIDs valid", Label: "UIDVALIDITY", Fields: []Field{"UIDVALIDITY", uint32(3857529045)}}},
		{`* FLAGS (\Answered \Flagged \Deleted \Seen \Draft)`,
			&Response{Tag: "*", Type: Data, Label: "FLAGS", Fields: []Field{"FLAGS", []Field{`\Answered`, `\Flagged`, `\Deleted`, `\Seen`, `\Draft`}}}},
		{`* OK [PERMANENTFLAGS (\Deleted \Seen \*)] Limited`,
			&Response{Tag: "*", Type: Status, Status: OK, Info: "Limited", Label: "PERMANENTFLAGS", Fields: []Field{"PERMANENTFLAGS", []Field{`\Deleted`, `\Seen`, `\*`}}}},
		{`A142 OK [READ-WRITE] SELECT completed`,
			&Response{Tag: "A142", Type: Done, Status: OK, Info: "SELECT completed", Label: "READ-WRITE", Fields: []Field{"READ-WRITE"}}},

		// Page 34
		{`* OK [PERMANENTFLAGS ()] No permanent flags permitted`,
			&Response{Tag: "*", Type: Status, Status: OK, Info: "No permanent flags permitted", Label: "PERMANENTFLAGS", Fields: []Field{"PERMANENTFLAGS", []Field(nil)}}},

		// Page 36
		{`* LIST () "/" blurdybloop`,
			&Response{Tag: "*", Type: Data, Label: "LIST", Fields: []Field{"LIST", []Field(nil), `"/"`, "blurdybloop"}}},
		{`* LIST (\Noselect) "/" foo`,
			&Response{Tag: "*", Type: Data, Label: "LIST", Fields: []Field{"LIST", []Field{`\Noselect`}, `"/"`, "foo"}}},
		{`* LIST () "/" foo/bar`,
			&Response{Tag: "*", Type: Data, Label: "LIST", Fields: []Field{"LIST", []Field(nil), `"/"`, "foo/bar"}}},

		// Page 44
		{`* LSUB () "." #news.comp.mail.mime`,
			&Response{Tag: "*", Type: Data, Label: "LSUB", Fields: []Field{"LSUB", []Field(nil), `"."`, "#news.comp.mail.mime"}}},
		{`* LSUB (\NoSelect) "." #news.comp.mail`,
			&Response{Tag: "*", Type: Data, Label: "LSUB", Fields: []Field{"LSUB", []Field{`\Noselect`}, `"."`, "#news.comp.mail"}}},

		// Page 45
		{`* STATUS blurdybloop (MESSAGES 231 UIDNEXT 44292)`,
			&Response{Tag: "*", Type: Data, Label: "STATUS", Fields: []Field{"STATUS", "blurdybloop", []Field{"MESSAGES", uint32(231), "UIDNEXT", uint32(44292)}}}},

		// Page 54
		{`* SEARCH 2 84 882`,
			&Response{Tag: "*", Type: Data, Label: "SEARCH", Fields: []Field{"SEARCH", uint32(2), uint32(84), uint32(882)}}},

		// Extra space at the end sent by Yahoo servers (violates RFC 3501)
		{`* SEARCH 2 84 882 `,
			&Response{Tag: "*", Type: Data, Label: "SEARCH", Fields: []Field{"SEARCH", uint32(2), uint32(84), uint32(882)}}},

		// Page 66
		{`* OK [ALERT] System shutdown in 10 minutes`,
			&Response{Tag: "*", Type: Status, Status: OK, Info: "System shutdown in 10 minutes", Label: "ALERT", Fields: []Field{"ALERT"}}},

		// Page 79
		{`* 23 FETCH (FLAGS (\Seen) RFC822.SIZE 44827)`,
			&Response{Tag: "*", Type: Data, Label: "FETCH", Fields: []Field{uint32(23), "FETCH", []Field{"FLAGS", []Field{`\Seen`}, "RFC822.SIZE", uint32(44827)}}}},

		// Page 80
		{`* 12 FETCH (FLAGS (\Seen) INTERNALDATE "17-Jul-1996 02:44:25 -0700"` +
			` RFC822.SIZE 4286 ENVELOPE ("Wed, 17 Jul 1996 02:23:25 -0700 (PDT)"` +
			` "IMAP4rev1 WG mtg summary and minutes"` +
			` (("Terry Gray" NIL "gray" "cac.washington.edu"))` +
			` (("Terry Gray" NIL "gray" "cac.washington.edu"))` +
			` (("Terry Gray" NIL "gray" "cac.washington.edu"))` +
			` ((NIL NIL "imap" "cac.washington.edu"))` +
			` ((NIL NIL "minutes" "CNRI.Reston.VA.US")` +
			` ("John Klensin" NIL "KLENSIN" "MIT.EDU")) NIL NIL` +
			` "<B27397-0100000@cac.washington.edu>")` +
			` BODY ("TEXT" "PLAIN" ("CHARSET" "US-ASCII") NIL NIL "7BIT" 3028` +
			` 92))`,
			&Response{Tag: "*", Type: Data, Label: "FETCH", Fields: []Field{
				uint32(12), "FETCH", []Field{"FLAGS", []Field{`\Seen`}, "INTERNALDATE", `"17-Jul-1996 02:44:25 -0700"`,
					"RFC822.SIZE", uint32(4286), "ENVELOPE", []Field{`"Wed, 17 Jul 1996 02:23:25 -0700 (PDT)"`,
						`"IMAP4rev1 WG mtg summary and minutes"`,
						[]Field{[]Field{`"Terry Gray"`, nil, `"gray"`, `"cac.washington.edu"`}},
						[]Field{[]Field{`"Terry Gray"`, nil, `"gray"`, `"cac.washington.edu"`}},
						[]Field{[]Field{`"Terry Gray"`, nil, `"gray"`, `"cac.washington.edu"`}},
						[]Field{[]Field{nil, nil, `"imap"`, `"cac.washington.edu"`}},
						[]Field{[]Field{nil, nil, `"minutes"`, `"CNRI.Reston.VA.US"`},
							[]Field{`"John Klensin"`, nil, `"KLENSIN"`, `"MIT.EDU"`}}, nil, nil,
						`"<B27397-0100000@cac.washington.edu>"`},
					"BODY", []Field{`"TEXT"`, `"PLAIN"`, []Field{`"CHARSET"`, `"US-ASCII"`}, nil, nil, `"7BIT"`, uint32(3028),
						uint32(92)}}},
			}},
		{`* 12 FETCH (BODY[HEADER] {342}` + CRLF + header + `)`,
			&Response{Tag: "*", Type: Data, Label: "FETCH", Fields: []Field{uint32(12), "FETCH", []Field{"BODY[HEADER]", lit(header)}}}},

		// Literals in BODY[...] are handled, but are not included in Fields
		{`* 12 FETCH (BODY[HEADER.FIELDS.NOT ({4}` + CRLF + `Date)]<0> NIL)`,
			&Response{Tag: "*", Type: Data, Label: "FETCH", Fields: []Field{uint32(12), "FETCH", []Field{"BODY[HEADER.FIELDS.NOT ({4})]<0>", nil}}}},

		// body-type-mpart is 1*body without a space in between
		{`* 11603 FETCH (BODYSTRUCTURE (` +
			`("TEXT" "PLAIN" ("CHARSET" "UTF-8") "text-body" NIL "7BIT" 1166 15 NIL NIL NIL)` +
			`("TEXT" "HTML" ("CHARSET" "UTF-8") "html-body" NIL "QUOTED-PRINTABLE" 15038 192 NIL NIL NIL)` +
			` "ALTERNATIVE" ("BOUNDARY" "----=_Part_169081_1994397778.1378998415121") NIL NIL))`,
			&Response{Tag: "*", Type: Data, Label: "FETCH", Fields: []Field{
				uint32(11603), "FETCH", []Field{"BODYSTRUCTURE", []Field{
					[]Field{`"TEXT"`, `"PLAIN"`, []Field{`"CHARSET"`, `"UTF-8"`}, `"text-body"`, nil, `"7BIT"`, uint32(1166), uint32(15), nil, nil, nil},
					[]Field{`"TEXT"`, `"HTML"`, []Field{`"CHARSET"`, `"UTF-8"`}, `"html-body"`, nil, `"QUOTED-PRINTABLE"`, uint32(15038), uint32(192), nil, nil, nil},
					`"ALTERNATIVE"`, []Field{`"BOUNDARY"`, `"----=_Part_169081_1994397778.1378998415121"`}, nil, nil}}},
			}},
	}
	c, s := newTestConn(1024)
	C := newTransport(c, nil)
	r := newReader(C, MemoryReader{}, "A")

	for _, test := range tests {
		C.clear()
		s.Write([]byte(test.in + CRLF))

		raw, err := r.Next()
		if raw == nil || err != nil {
			t.Errorf("Next(%+q) unexpected error; %v", test.in, err)
			continue
		}
		out, err := raw.Parse()
		if out != nil {
			out.Order = 0
			out.Raw = nil
			out.Literals = nil
		}
		if test.out == nil {
			if err == nil {
				t.Errorf("Parse(%+q) expected error; got\n%#v", test.in, out)
			}
		} else if err != nil {
			t.Errorf("Parse(%+q) unexpected error; %v", test.in, err)
		} else if !reflect.DeepEqual(out, test.out) {
			t.Errorf("Parse(%+q) expected\n%#v; got\n%#v", test.in, test.out, out)
		}
	}
}
