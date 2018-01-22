// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import "testing"

var encode = []struct {
	in  string
	out string
	ok  bool
}{
	// Printable ASCII
	{"", "", true},
	{"a", "a", true},
	{"ab", "ab", true},
	{"-", "-", true},
	{"&", "&-", true},
	{"&&", "&-&-", true},
	{"&&&-&", "&-&-&--&-", true},
	{"-&*&-", "-&-*&--", true},
	{"a&b", "a&-b", true},
	{"a&", "a&-", true},
	{"&b", "&-b", true},
	{"-a&", "-a&-", true},
	{"&b-", "&-b-", true},

	// Unicode range
	{"\u0000", "&AAA-", true},
	{"\n", "&AAo-", true},
	{"\r", "&AA0-", true},
	{"\u001F", "&AB8-", true},
	{"\u0020", " ", true},
	{"\u0025", "%", true},
	{"\u0026", "&-", true},
	{"\u0027", "'", true},
	{"\u007E", "~", true},
	{"\u007F", "&AH8-", true},
	{"\u0080", "&AIA-", true},
	{"\u00FF", "&AP8-", true},
	{"\u07FF", "&B,8-", true},
	{"\u0800", "&CAA-", true},
	{"\uFFEF", "&,+8-", true},
	{"\uFFFF", "&,,8-", true},
	{"\U00010000", "&2ADcAA-", true},
	{"\U0010FFFF", "&2,,f,w-", true},

	// Padding
	{"\x00\x1F", "&AAAAHw-", true},                         // 2
	{"\x00\x1F\x7F", "&AAAAHwB,-", true},                   // 0
	{"\x00\x1F\x7F\u0080", "&AAAAHwB,AIA-", true},          // 1
	{"\x00\x1F\x7F\u0080\u00FF", "&AAAAHwB,AIAA,w-", true}, // 2

	// Mix
	{"a\x00", "a&AAA-", true},
	{"\x00a", "&AAA-a", true},
	{"&\x00", "&-&AAA-", true},
	{"\x00&", "&AAA-&-", true},
	{"a\x00&", "a&AAA-&-", true},
	{"a&\x00", "a&-&AAA-", true},
	{"&a\x00", "&-a&AAA-", true},
	{"&\x00a", "&-&AAA-a", true},
	{"\x00&a", "&AAA-&-a", true},
	{"\x00a&", "&AAA-a&-", true},
	{"ab&\uFFFF", "ab&-&,,8-", true},
	{"a&b\uFFFF", "a&-b&,,8-", true},
	{"&ab\uFFFF", "&-ab&,,8-", true},
	{"ab\uFFFF&", "ab&,,8-&-", true},
	{"a\uFFFFb&", "a&,,8-b&-", true},
	{"\uFFFFab&", "&,,8-ab&-", true},

	{"\x20\x25&\x27\x7E", " %&-'~", true},
	{"\x1F\x20&\x7E\x7F", "&AB8- &-~&AH8-", true},
	{"&\x00\x19\x7F\u0080", "&-&AAAAGQB,AIA-", true},
	{"\x00&\x19\x7F\u0080", "&AAA-&-&ABkAfwCA-", true},
	{"\x00\x19&\x7F\u0080", "&AAAAGQ-&-&AH8AgA-", true},
	{"\x00\x19\x7F&\u0080", "&AAAAGQB,-&-&AIA-", true},
	{"\x00\x19\x7F\u0080&", "&AAAAGQB,AIA-&-", true},
	{"&\x00\x1F\x7F\u0080", "&-&AAAAHwB,AIA-", true},
	{"\x00&\x1F\x7F\u0080", "&AAA-&-&AB8AfwCA-", true},
	{"\x00\x1F&\x7F\u0080", "&AAAAHw-&-&AH8AgA-", true},
	{"\x00\x1F\x7F&\u0080", "&AAAAHwB,-&-&AIA-", true},
	{"\x00\x1F\x7F\u0080&", "&AAAAHwB,AIA-&-", true},

	// Russian
	{"\u041C\u0430\u043A\u0441\u0438\u043C \u0425\u0438\u0442\u0440\u043E\u0432",
		"&BBwEMAQ6BEEEOAQ8- &BCUEOARCBEAEPgQy-", true},

	// RFC 3501
	{"~peter/mail/\u53F0\u5317/\u65E5\u672C\u8A9E", "~peter/mail/&U,BTFw-/&ZeVnLIqe-", true},
	{"~peter/mail/\u53F0\u5317/\u65E5\u672C\u8A9E", "~peter/mail/&U,BTFw-/&ZeVnLIqe-", true},
	{"\u263A!", "&Jjo-!", true},
	{"\u53F0\u5317\u65E5\u672C\u8A9E", "&U,BTF2XlZyyKng-", true},

	// RFC 2152 (modified)
	{"\u0041\u2262\u0391\u002E", "A&ImIDkQ-.", true},
	{"Hi Mom -\u263A-!", "Hi Mom -&Jjo--!", true},
	{"\u65E5\u672C\u8A9E", "&ZeVnLIqe-", true},

	// 8->16 and 24->16 byte UTF-8 to UTF-16 conversion
	{"\u0000\u0001\u0002\u0003\u0004\u0005\u0006\u0007", "&AAAAAQACAAMABAAFAAYABw-", true},
	{"\u0800\u0801\u0802\u0803\u0804\u0805\u0806\u0807", "&CAAIAQgCCAMIBAgFCAYIBw-", true},

	// Invalid UTF-8 (bad bytes are converted to U+FFFD)
	{"\xC0\x80", "&,,3,,Q-", false},                     // U+0000
	{"\xF4\x90\x80\x80", "&,,3,,f,9,,0-", false},        // U+110000
	{"\xF7\xBF\xBF\xBF", "&,,3,,f,9,,0-", false},        // U+1FFFFF
	{"\xF8\x88\x80\x80\x80", "&,,3,,f,9,,3,,Q-", false}, // U+200000
	{"\xF4\x8F\xBF\x3F", "&,,3,,f,9-?", false},          // U+10FFFF (bad byte)
	{"\xF4\x8F\xBF", "&,,3,,f,9-", false},               // U+10FFFF (short)
	{"\xF4\x8F", "&,,3,,Q-", false},
	{"\xF4", "&,,0-", false},
	{"\x00\xF4\x00", "&AAD,,QAA-", false},
}

var decode = []struct {
	in  string
	out string
	ok  bool
}{
	// Basics (the inverse test on encode checks other valid inputs)
	{"", "", true},
	{"abc", "abc", true},
	{"&-abc", "&abc", true},
	{"abc&-", "abc&", true},
	{"a&-b&-c", "a&b&c", true},
	{"&ABk-", "\x19", true},
	{"&AB8-", "\x1F", true},
	{"ABk-", "ABk-", true},
	{"&-,&-&AP8-&-", "&,&\u00FF&", true},
	{"&-&-,&AP8-&-", "&&,\u00FF&", true},
	{"abc &- &AP8A,wD,- &- xyz", "abc & \u00FF\u00FF\u00FF & xyz", true},

	// Illegal code point in ASCII
	{"\x00", "", false},
	{"\x1F", "", false},
	{"abc\n", "", false},
	{"abc\x7Fxyz", "", false},
	{"\uFFFD", "", false},
	{"\u041C", "", false},

	// Invalid Base64 alphabet
	{"&/+8-", "", false},
	{"&*-", "", false},
	{"&ZeVnLIqe -", "", false},

	// CR and LF in Base64
	{"&ZeVnLIqe\r\n-", "", false},
	{"&ZeVnLIqe\r\n\r\n-", "", false},
	{"&ZeVn\r\n\r\nLIqe-", "", false},

	// Padding not stripped
	{"&AAAAHw=-", "", false},
	{"&AAAAHw==-", "", false},
	{"&AAAAHwB,AIA=-", "", false},
	{"&AAAAHwB,AIA==-", "", false},

	// One byte short
	{"&2A-", "", false},
	{"&2ADc-", "", false},
	{"&AAAAHwB,A-", "", false},
	{"&AAAAHwB,A=-", "", false},
	{"&AAAAHwB,A==-", "", false},
	{"&AAAAHwB,A===-", "", false},
	{"&AAAAHwB,AI-", "", false},
	{"&AAAAHwB,AI=-", "", false},
	{"&AAAAHwB,AI==-", "", false},

	// Implicit shift
	{"&", "", false},
	{"&Jjo", "", false},
	{"Jjo&", "", false},
	{"&Jjo&", "", false},
	{"&Jjo!", "", false},
	{"&Jjo+", "", false},
	{"abc&Jjo", "", false},

	// Null shift
	{"&AGE-&Jjo-", "", false},
	{"&U,BTFw-&ZeVnLIqe-", "", false},

	// ASCII in Base64
	{"&AGE-", "", false},            // "a"
	{"&ACY-", "", false},            // "&"
	{"&AGgAZQBsAGwAbw-", "", false}, // "hello"
	{"&JjoAIQ-", "", false},         // "\u263a!"

	// Bad surrogate
	{"&2AA-", "", false},    // U+D800
	{"&2AD-", "", false},    // U+D800
	{"&3AA-", "", false},    // U+DC00
	{"&2AAAQQ-", "", false}, // U+D800 'A'
	{"&2AD,,w-", "", false}, // U+D800 U+FFFF
	{"&3ADYAA-", "", false}, // U+DC00 U+D800
}

func TestUTF7Encode(t *testing.T) {
	for _, test := range encode {
		out := UTF7Encode(test.in)
		if out != test.out {
			t.Errorf("UTF7Encode(%+q) expected %+q; got %+q", test.in, test.out, out)
		}
	}
}

func TestUTF7Decode(t *testing.T) {
	for _, test := range decode {
		out, err := UTF7Decode(test.in)
		if out != test.out {
			t.Errorf("UTF7Decode(%+q) expected %+q; got %+q", test.in, test.out, out)
		}
		if test.ok {
			if err != nil {
				t.Errorf("UTF7Decode(%+q) unexpected error; %v", test.in, err)
			}
		} else if err == nil {
			t.Errorf("UTF7Decode(%+q) expected error", test.in)
		}
	}
}

func TestUTF7Inverse(t *testing.T) {
	for _, test := range encode {
		if test.ok {
			in, _ := UTF7Decode(UTF7Encode(test.in))
			if in != test.in {
				t.Errorf("EncodeInverse(%+q) expected %+q; got %+q", test.in, test.in, in)
			}
		}
	}
	for _, test := range decode {
		if test.ok {
			out, _ := UTF7Decode(test.in)
			in := UTF7Encode(out)
			if in != test.in {
				t.Errorf("DecodeInverse(%+q) expected %+q; got %+q", test.in, test.in, in)
			}
		}
	}
}
