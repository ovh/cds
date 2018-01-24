// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"strings"
	"testing"
)

func q(s string) string  { return `"` + s + `"` }
func uq(s string) string { return `*"` + s + `"` }

var quote_tests = []struct {
	in   string
	utf8 bool
	out  string
}{
	// Invalid
	{"\x00", false, ""},
	{"\x00", true, ""},
	{"\r", false, ""},
	{"\r", true, ""},
	{"\n", false, ""},
	{"\n", true, ""},
	{"\x80", false, ""},
	{"\x80", true, ""},
	{"\xFF", false, ""},
	{"\xFF", true, ""},

	// ASCII min, max
	{"\x01", false, q("\x01")},
	{"\x01", true, q("\x01")},
	{"\x7F", false, q("\x7F")},
	{"\x7F", true, q("\x7F")},

	// Valid
	{``, false, q(``)},
	{`a`, false, q(`a`)},
	{`ab`, false, q(`ab`)},
	{`abc`, true, q(`abc`)},
	{`"`, false, q(`\"`)},
	{`\`, false, q(`\\`)},
	{`""`, false, q(`\"\"`)},
	{`\\`, false, q(`\\\\`)},
	{`"a"`, false, q(`\"a\"`)},
	{`"\"`, false, q(`\"\\\"`)},
	{`"""`, false, q(`\"\"\"`)},
	{`\"\`, false, q(`\\\"\\`)},
	{`"\abc\"`, false, q(`\"\\abc\\\"`)},
	{`"\abc\"`, true, q(`\"\\abc\\\"`)},
	{`hello, world`, false, q(`hello, world`)},
	{`\hello/,/world\`, false, q(`\\hello/,/world\\`)},

	// Unicode
	{"\u65e5", false, ""},
	{"\u65e5", true, uq("\u65e5")},
	{`"\u65e5"`, false, q(`\"\\u65e5\"`)},
	{`"\u65e5"`, true, q(`\"\\u65e5\"`)},

	{q("\u65e5"), false, ""},
	{q("\u65e5"), true, uq(`\"` + "\u65e5" + `\"`)},
	{q("\u65e5\u672c\u8a9e!"), false, ""},
	{q("\u65e5\u672c\u8a9e!"), true, uq(`\"` + "\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e!" + `\"`)},

	{"\xe6\x97", true, ""},
	{"\xe6\x97\\", true, ""},
	{"\xe6\x97\x00", true, ""},
	{"\xe6\x97\x7f", true, ""},
	{"\xe6\xff\x97\xa5", true, ""},
	{"\xe6\x97\xa5", true, uq("\u65e5")},
}

var unquote_tests = []struct {
	in  string
	out string
	ok  bool
}{
	// Invalid
	{``, ``, false},
	{`"`, ``, false},
	{`" `, ``, false},
	{` "`, ``, false},
	{`''`, ``, false},
	{`*"`, ``, false},
	{`*" `, ``, false},

	{q(`\`), ``, false},
	{uq(`\`), ``, false},
	{q(`"`), ``, false},
	{uq(`"`), ``, false},
	{q(`\\\`), ``, false},
	{uq(`\\\`), ``, false},

	{q("\x00"), ``, false},
	{q("\r"), ``, false},
	{q("\n"), ``, false},
	{q("\x80"), ``, false},
	{q("\xFF"), ``, false},

	// Valid
	{q(""), "", true},
	{uq(""), "", true},
	{q(" "), " ", true},
	{uq(" "), " ", true},
	{q("'"), "'", true},
	{uq("'"), "'", true},
	{q("abc"), "abc", true},
	{uq("abc"), "abc", true},

	{q("\x01"), "\x01", true},
	{q("\x7F"), "\x7F", true},

	{q(`\\`), `\`, true},
	{q(`\"`), `"`, true},
	{q(`\\\\`), `\\`, true},
	{q(`\"\\`), `"\`, true},
	{q(`\\\"`), `\"`, true},
	{q(`\\\\\\`), `\\\`, true},
	{q(`\\\"\\`), `\"\`, true},

	// Unicode
	{q("\u65e5"), "\u65e5", true},
	{uq("\u65e5"), "\u65e5", true},

	{q("/\u65e5\u672c\u8a9e\\"), "", false},
	{q("/\u65e5\u672c\u8a9e\\\\"), "/\u65e5\u672c\u8a9e\\", true},
	{uq("\u65e5\u672c\u8a9e!"), "\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e!", true},

	{q("\xe6\x97"), "", false},
	{uq("\xe6\x97"), "", false},
	{uq("\xe6\x97\\"), "", false},
	{uq("\xe6\x97\x00"), "", false},
	{uq("\xe6\x97\x7f"), "", false},
	{uq("\xe6\xff\x97\xa5"), "", false},
	{uq("\xe6\x97\xa5"), "\u65e5", true},
	{q("\xe6\x97\xa5"), "\u65e5", true},
}

func TestStringsQuote(t *testing.T) {
	for _, test := range quote_tests {
		out := Quote(test.in, test.utf8)
		if out != test.out {
			t.Errorf("Quote(%#q, %v) expected %#q; got %#q", test.in, test.utf8, test.out, out)
		}
		if test.out != "" {
			if !Quoted(out) || !Quoted([]byte(out)) {
				t.Errorf("Quoted(Quote(%#q)) expected true", test.in)
			} else if utf8 := test.out[0] == '*'; QuotedUTF8(out) != utf8 || QuotedUTF8([]byte(out)) != utf8 {
				t.Errorf("QuotedUTF8(Quote(%#q)) expected %v", test.in, utf8)
			}
		}
	}
}

func TestStringsUnquote(t *testing.T) {
	for _, test := range unquote_tests {
		out, ok := Unquote(test.in)
		if out != test.out || ok != test.ok {
			t.Errorf("Unquote(%#q) expected %#q (%v); got %#q (%v)", test.in, test.out, test.ok, out, ok)
		}
	}
}

func TestStringsQuoteInverse(t *testing.T) {
	for _, test := range quote_tests {
		if test.out == "" {
			continue
		}
		in, ok := Unquote(Quote(test.in, test.utf8))
		if in != test.in || !ok {
			t.Errorf("QuoteInverse(%#q) expected %#q (true); got %#q (%v)", test.in, test.in, in, ok)
		}
	}
	for _, test := range unquote_tests {
		if !test.ok {
			continue
		}
		out, ok := Unquote(test.in)
		out, ok = Unquote(Quote(out, true))
		if out != test.out || !ok {
			t.Errorf("UnquoteInverse(%#q) expected %#q (true); got %#q (%v)", test.in, test.out, out, ok)
		}
	}
}

func TestStringsToUpper(t *testing.T) {
	// Handling of ASCII bytes is identical to strings.ToUpper
	for _, in := range []string{"TEST", "test", "Test", "TeSt", "TesT", "tESt"} {
		exp := strings.ToUpper(in)
		out := toUpper(in)
		if out != exp {
			t.Errorf("toUpper(%+q) expected %+q; got %+q", in, exp, out)
		}
	}
	for in := byte(0); in < char; in++ {
		exp := strings.ToUpper(string(in))
		out := toUpper(string(in))
		if out != exp {
			t.Fatalf("toUpper(%+q) expected %+q; got %+q", string(in), exp, out)
		}
	}
	// Non-ASCII bytes are unaffected
	for in := byte(char); in != 0; in++ {
		exp := string(in)
		out := toUpper(string(in))
		if out != exp {
			t.Fatalf("toUpper(%+q) expected %+q; got %+q", string(in), exp, out)
		}
	}
}
