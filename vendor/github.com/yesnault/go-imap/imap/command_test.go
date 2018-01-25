// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"reflect"
	"testing"
	"time"
)

func newSeqSet(set string) *SeqSet {
	s, _ := NewSeqSet(set)
	return s
}

func TestCommand(t *testing.T) {
	tests := []struct {
		tag    string
		name   string
		fields []Field
		out    *Command
	}{
		{"A001", "CAPABILITY", nil, &Command{
			name: "CAPABILITY",
			tag:  "A001",
			raw:  `A001 CAPABILITY`}},

		{"", "setCaps", []Field{"IMAP4rev1"}, nil},
		{"A001", "LOGIN", []Field{`"username"`, `"password"`}, &Command{
			name: "LOGIN",
			tag:  "A001",
			raw:  `A001 LOGIN "username" "password"`}},
		{"A002", "LOGIN", []Field{lit(`username`), `"password"`}, &Command{
			name: "LOGIN",
			tag:  "A002",
			raw:  `A002 LOGIN {8} "password"`}},
		{"A003", "LOGIN", []Field{`"username"`, lit(`password`)}, &Command{
			name: "LOGIN",
			tag:  "A003",
			raw:  `A003 LOGIN "username" {8}`}},
		{"A004", "LOGIN", []Field{lit(`username`), lit(`password`)}, &Command{
			name: "LOGIN",
			tag:  "A004",
			raw:  `A004 LOGIN {8} {8}`}},

		{"", "setCaps", []Field{"IMAP4rev1", "LITERAL+"}, nil},
		{"A005", "LOGIN", []Field{lit(`username`), lit(`password`)}, &Command{
			name: "LOGIN",
			tag:  "A005",
			raw:  `A005 LOGIN {8+} {8+}`}},

		{"", "setCaps", []Field{"IMAP4rev1", "LITERAL+", "BINARY"}, nil},
		{"A006", "LOGIN", []Field{lit(`username`), lit8(`password`)}, &Command{
			name: "LOGIN",
			tag:  "A006",
			raw:  `A006 LOGIN {8+} ~{8+}`}},

		{"A001", "FETCH", []Field{newSeqSet("1,2,3,4"), []Field{"FAST"}}, &Command{
			name:   "FETCH",
			seqset: newSeqSet("1:4"),
			tag:    "A001",
			raw:    `A001 FETCH 1:4 (FAST)`}},

		{"A001", "UID FETCH", []Field{newSeqSet("1,3:*"), []Field{"BODY[]", "UID"}}, &Command{
			uid:    true,
			name:   "FETCH",
			seqset: newSeqSet("1,3:*"),
			tag:    "A001",
			raw:    `A001 UID FETCH 1,3:* (BODY[] UID)`}},

		{"A001", "CHECK", []Field{"str", 123, []Field{[]Field(nil), []byte("data")}, nil, NewFlagSet(`\Answered`, `\Flagged`)}, &Command{
			name: "CHECK",
			tag:  "A001",
			raw:  `A001 CHECK str 123 (() data) NIL (\Answered \Flagged)`}},
		{"A002", "CHECK", []Field{time.Date(1986, time.February, 1, 23, 0, 1, 0, time.UTC)}, &Command{
			name: "CHECK",
			tag:  "A002",
			raw:  `A002 CHECK " 1-Feb-1986 23:00:01 +0000"`}},
	}
	c := &Client{
		Caps:          make(map[string]bool),
		CommandConfig: defaultCommands(),
		debugLog:      newDebugLog(nil, LogNone),
	}
	for _, test := range tests {
		if test.name == "setCaps" {
			c.setCaps(test.fields)
			continue
		}
		out := newCommand(c, test.name)
		if out != nil {
			out.config = CommandConfig{}
		}
		if test.out != nil {
			test.out.client = c
		}
		if _, err := out.build(test.tag, test.fields); err != nil {
			t.Errorf("build(%s %s) unexpected error; %v", test.tag, test.name, err)
		} else if !reflect.DeepEqual(out, test.out) {
			t.Errorf("build(%s %s) expected\n%#v; got\n%#v", test.tag, test.name, test.out, out)
		}
	}
}
