// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

type xlit struct{ Literal }

func fname(v reflect.Value) (name string) {
	name = runtime.FuncForPC(v.Pointer()).Name()
	if i := strings.LastIndex(name, "."); i != -1 {
		name = name[i+1:]
	}
	return name
}

func TestField(t *testing.T) {
	tests := []struct {
		call interface{}
		in   Field
		out  interface{}
	}{
		{TypeOf, nil, NIL},
		{TypeOf, ``, FieldType(0)},
		{TypeOf, 42, FieldType(0)},

		{TypeOf, `x`, Atom},
		{TypeOf, `FETCH`, Atom},
		{TypeOf, `\Seen`, Atom},

		{TypeOf, uint32(0), Number},
		{TypeOf, uint32(1), Number},
		{TypeOf, ^uint32(0), Number},

		{TypeOf, `""`, QuotedString},
		{TypeOf, `*""`, QuotedString},
		{TypeOf, `"x"`, QuotedString},
		{TypeOf, `*"x"`, QuotedString},
		{TypeOf, `"42"`, QuotedString},
		{TypeOf, `"\"`, QuotedString},
		{TypeOf, `"\x"`, QuotedString},

		{TypeOf, (*literal)(nil), LiteralString},
		{TypeOf, lit(``), LiteralString},
		{TypeOf, lit(`x`), LiteralString},
		{TypeOf, lit8(``), LiteralString},
		{TypeOf, lit8(`x`), LiteralString},
		{TypeOf, xlit{lit(``)}, LiteralString},
		{TypeOf, xlit{lit(`x`)}, LiteralString},
		{TypeOf, xlit{lit8(``)}, LiteralString},
		{TypeOf, xlit{lit8(`x`)}, LiteralString},

		{TypeOf, []Field(nil), List},
		{TypeOf, []Field{``}, List},
		{TypeOf, []Field{`x`}, List},
		{TypeOf, []Field{`\Seen`, `\Flagged`}, List},

		{TypeOf, []byte(nil), Bytes},
		{TypeOf, []byte(``), Bytes},
		{TypeOf, []byte(`x`), Bytes},

		{AsAtom, nil, ``},
		{AsAtom, ``, ``},
		{AsAtom, 42, ``},
		{AsAtom, `""`, ``},
		{AsAtom, `"x"`, ``},
		{AsAtom, `*"x"`, ``},
		{AsAtom, lit(`x`), ``},
		{AsAtom, []byte(`x`), ``},
		{AsAtom, `x`, `x`},
		{AsAtom, `FETCH`, `FETCH`},
		{AsAtom, `\seen`, `\seen`},

		{AsNumber, nil, uint32(0)},
		{AsNumber, 0, uint32(0)},
		{AsNumber, 1, uint32(0)},
		{AsNumber, ``, uint32(0)},
		{AsNumber, `1`, uint32(0)},
		{AsNumber, []byte{1}, uint32(0)},
		{AsNumber, uint32(0), uint32(0)},
		{AsNumber, uint32(1), uint32(1)},
		{AsNumber, ^uint32(0), ^uint32(0)},

		{AsString, nil, ``},
		{AsString, ``, ``},
		{AsString, `"\"`, ``},
		{AsString, `"\x"`, ``},
		{AsString, []byte(nil), ``},
		{AsString, []byte(`x`), ``},
		{AsString, []byte(`"x"`), ``},
		{AsString, `x`, `x`},
		{AsString, `""`, ``},
		{AsString, `*""`, ``},
		{AsString, `"x"`, `x`},
		{AsString, `"\""`, `"`},
		{AsString, `*"x"`, `x`},
		{AsString, lit(``), ``},
		{AsString, lit(`x`), `x`},
		{AsString, lit(`"`), `"`},
		{AsString, lit8(``), ``},
		{AsString, lit8(`x`), `x`},
		{AsString, lit8(`x\"`), `x\"`},
		{AsString, xlit{lit(``)}, ``},
		{AsString, xlit{lit("\x00\r\n")}, "\x00\r\n"},

		{AsBytes, nil, []byte(nil)},
		{AsBytes, ``, []byte(nil)},
		{AsBytes, `x`, []byte(nil)},
		{AsBytes, `"\"`, []byte(nil)},
		{AsBytes, `"\x"`, []byte(nil)},
		{AsBytes, []byte(nil), []byte(nil)},
		{AsBytes, []byte(`x`), []byte(`x`)},
		{AsBytes, []byte(`"x"`), []byte(`"x"`)},
		{AsBytes, `""`, []byte(nil)},
		{AsBytes, `*""`, []byte(nil)},
		{AsBytes, `"x"`, []byte(`x`)},
		{AsBytes, `"\""`, []byte(`"`)},
		{AsBytes, `*"x"`, []byte(`x`)},
		{AsBytes, lit(``), []byte(nil)},
		{AsBytes, lit(`x`), []byte(`x`)},
		{AsBytes, lit8(`x`), []byte(`x`)},
		{AsBytes, lit8(`\x00\r\n`), []byte(`\x00\r\n`)},
		{AsBytes, xlit{lit(``)}, []byte(nil)},
		{AsBytes, xlit{lit(`x`)}, []byte(`x`)},
		{AsBytes, xlit{lit8(`x`)}, []byte(`x`)},
		{AsBytes, xlit{lit8(`\x00\r\n`)}, []byte(`\x00\r\n`)},

		{AsList, nil, []Field(nil)},
		{AsList, ``, []Field(nil)},
		{AsList, []byte(`42`), []Field(nil)},
		{AsList, []Field(nil), []Field(nil)},
		{AsList, []Field{}, []Field{}},
		{AsList, []Field{`x`}, []Field{"x"}},
		{AsList, []Field{`\Seen`, `\Flagged`}, []Field{`\Seen`, `\Flagged`}},

		{AsDateTime, nil, time.Time{}},
		{AsDateTime, time.Now(), time.Time{}},
		{AsDateTime, ``, time.Time{}},
		{AsDateTime, `""`, time.Time{}},
		{AsDateTime, `"17-Jul-1996"`, time.Time{}},
		{AsDateTime, `"02:44:25"`, time.Time{}},
		{AsDateTime, `"17-Jul-1996 02:44:25"`, time.Time{}},
		{AsDateTime, `*"17-Jul-1996 02:44:25 -0700"`, time.Time{}},
		{AsDateTime, `"17-Jul-1996 02:44:25 -0700"`, time.Date(1996, time.July, 17, 2, 44, 25, 0, MST)},
		{AsDateTime, `"07-Jul-1996 02:44:25 -0700"`, time.Date(1996, time.July, 7, 2, 44, 25, 0, MST)},
		{AsDateTime, `" 7-Jul-1996  2:44:25 -0700"`, time.Date(1996, time.July, 7, 2, 44, 25, 0, MST)},
		{AsDateTime, `"7-Jul-1996 2:44:25 -0700"`, time.Date(1996, time.July, 7, 2, 44, 25, 0, MST)},
		{AsDateTime, `"7-Jul-1996 00:00:00 -0700"`, time.Date(1996, time.July, 7, 0, 0, 0, 0, MST)},
		{AsDateTime, `"7-Jul-1996 0:10:01 -0700"`, time.Date(1996, time.July, 7, 0, 10, 1, 0, MST)},

		{AsMailbox, nil, ``},
		{AsMailbox, ``, ``},
		{AsMailbox, `x`, `x`},
		{AsMailbox, `\"`, `\"`},
		{AsMailbox, `""`, ``},
		{AsMailbox, `"x"`, `x`},
		{AsMailbox, `"\""`, `"`},
		{AsMailbox, `&`, `&`},
		{AsMailbox, `&-`, `&`},
		{AsMailbox, `&x`, `&x`},
		{AsMailbox, `"&"`, `&`},
		{AsMailbox, `"&-"`, `&`},
		{AsMailbox, `"&x"`, `&x`},
		{AsMailbox, `*"&-"`, `&-`},
		{AsMailbox, lit(`&`), `&`},
		{AsMailbox, lit(`&-`), `&`},
		{AsMailbox, lit(`&x`), `&x`},
		{AsMailbox, lit(`"&-"`), `"&"`},
		{AsMailbox, lit(`*"&-"`), `*"&"`},
		{AsMailbox, lit(`&Jjo!`), `&Jjo!`},
		{AsMailbox, `inbox`, `INBOX`},
		{AsMailbox, `"iNbOx"`, `INBOX`},
		{AsMailbox, lit(`InBoX`), `INBOX`},
		{AsMailbox, lit(`"Inbox"`), `"Inbox"`},

		{AsFieldMap, nil, FieldMap(nil)},
		{AsFieldMap, uint32(1), FieldMap(nil)},
		{AsFieldMap, []byte(`xy`), FieldMap(nil)},
		{AsFieldMap, FieldMap{`x`: `y`}, FieldMap(nil)},
		{AsFieldMap, []Field{`x`}, FieldMap(nil)},
		{AsFieldMap, []Field{``, `x`}, FieldMap(nil)},
		{AsFieldMap, []Field{`x`, `y`, `z`}, FieldMap(nil)},
		{AsFieldMap, []Field{`"X"`, uint32(42)}, FieldMap(nil)},
		{AsFieldMap, []Field(nil), FieldMap{}},
		{AsFieldMap, []Field{}, FieldMap{}},
		{AsFieldMap, []Field{`X`, uint32(42)}, FieldMap{`X`: uint32(42)}},
		{AsFieldMap, []Field{`x`, `y`, `z`, nil}, FieldMap{`X`: `y`, `Z`: nil}},
		{AsFieldMap, []Field{`x`, `"y"`, `z`, []Field(nil)}, FieldMap{`X`: `"y"`, `Z`: []Field(nil)}},

		{AsFlagSet, nil, FlagSet(nil)},
		{AsFlagSet, uint32(1), FlagSet(nil)},
		{AsFlagSet, []byte(`\Seen`), FlagSet(nil)},
		{AsFlagSet, NewFlagSet(`\Seen`), FlagSet(nil)},
		{AsFlagSet, []Field{``}, FlagSet(nil)},
		{AsFlagSet, []Field{`"\Seen"`}, FlagSet(nil)},
		{AsFlagSet, []Field(nil), FlagSet{}},
		{AsFlagSet, []Field{}, FlagSet{}},
		{AsFlagSet, []Field{`x`}, FlagSet{`x`: true}},
		{AsFlagSet, []Field{`x`}, NewFlagSet(`x`)},
		{AsFlagSet, []Field{`x`, `y`}, NewFlagSet(`x`, `y`)},
		{AsFlagSet, []Field{`\Seen`, `\deleted`}, NewFlagSet(`\Seen`, `\deleted`)},
	}
	for _, test := range tests {
		call := reflect.ValueOf(test.call)
		args := []reflect.Value{reflect.ValueOf(&test.in).Elem()}
		out := call.Call(args)[0].Interface()
		if !reflect.DeepEqual(out, test.out) {
			t.Errorf("%s(%#v) expected %v; got %v", fname(call), test.in, test.out, out)
		}
	}
	in := lit("x")
	out := AsBytes(in)
	if reflect.ValueOf(out).Pointer() != reflect.ValueOf(in.(*literal).data).Pointer() {
		t.Errorf("AsBytes took the slow path for *literal")
	}
}
