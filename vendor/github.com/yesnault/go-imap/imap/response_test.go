// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"reflect"
	"testing"
	"time"
)

var (
	UTC = time.FixedZone("", 0)
	MST = time.FixedZone("", -7*60*60)
)

func TestResponseDecoders(t *testing.T) {
	tests := []struct {
		in   string
		call string
		out  interface{}
	}{
		// Original string
		{`A142 OK [READ-WRITE] SELECT completed`,
			"String", "A142 OK [READ-WRITE] SELECT completed"},

		// Numeric value -> uint32
		{`* STATUS blurdybloop (MESSAGES 231 UIDNEXT 44292)`,
			"Value", uint32(0)},
		{`* 172 EXISTS`,
			"Value", uint32(172)},
		{`* 1 RECENT`,
			"Value", uint32(1)},
		{`* 22 EXPUNGE`,
			"Value", uint32(22)},
		{`* OK [UNSEEN 12] Message 12 is first unseen`,
			"Value", uint32(12)},
		{`* OK [UIDNEXT 4392] Predicted next UID`,
			"Value", uint32(4392)},
		{`* OK [UIDVALIDITY 3857529045] UIDs valid`,
			"Value", uint32(3857529045)},

		// Authentication challenge -> []byte
		{`+ Welcome!`,
			"Challenge", []byte(nil)},
		{`+ YDMGCSqGSIb3EgECAgIBAAD/////6jcyG4GE3KkTzBeBiVHeceP2CWY0SR0fAQAgAAQEBAQ=`,
			"Challenge", []byte("\x60\x33\x06\x09\x2A\x86\x48\x86\xF7\x12\x01" +
				"\x02\x02\x02\x01\x00\x00\xFF\xFF\xFF\xFF\xEA\x37\x32\x1B\x81" +
				"\x84\xDC\xA9\x13\xCC\x17\x81\x89\x51\xDE\x71\xE3\xF6\x09\x66" +
				"\x34\x49\x1D\x1F\x01\x00\x20\x00\x04\x04\x04\x04")},

		// LIST and LSUB -> MailboxInfo
		{`* NOT LIST`,
			"MailboxInfo", (*MailboxInfo)(nil)},
		{`* LIST () NIL ""`,
			"MailboxInfo", &MailboxInfo{
				Attrs: NewFlagSet()}},
		{`* LIST () "\\" iNbOx`,
			"MailboxInfo", &MailboxInfo{
				Attrs: NewFlagSet(),
				Delim: `\`,
				Name:  "INBOX"}},
		{`* LSUB () "/" blurdybloop`,
			"MailboxInfo", &MailboxInfo{
				Attrs: NewFlagSet(),
				Delim: "/",
				Name:  "blurdybloop"}},
		{`* LSUB () "/" [blurdybloop]`,
			"MailboxInfo", &MailboxInfo{
				Attrs: NewFlagSet(),
				Delim: "/",
				Name:  "[blurdybloop]"}},
		{`* LIST (\Noselect) "." "#foo.bar"`,
			"MailboxInfo", &MailboxInfo{
				Attrs: NewFlagSet(`\Noselect`),
				Delim: ".",
				Name:  "#foo.bar"}},
		{`* LIST (\Noselect) "/" #foo.bar/[blurdybloop]`,
			"MailboxInfo", &MailboxInfo{
				Attrs: NewFlagSet(`\Noselect`),
				Delim: "/",
				Name:  "#foo.bar/[blurdybloop]"}},
		{`* LIST (\NoInferiors \NoSelect) NIL {6}` + CRLF + `foobar`,
			"MailboxInfo", &MailboxInfo{
				Attrs: NewFlagSet(`\Noselect`, `\Noinferiors`),
				Delim: "",
				Name:  "foobar"}},
		{`* LSUB (\noselect \marked) "/" ~peter/mail/&U,BTFw-/&ZeVnLIqe-`,
			"MailboxInfo", &MailboxInfo{
				Attrs: NewFlagSet(`\Noselect`, `\Marked`),
				Delim: "/",
				Name:  "~peter/mail/\u53F0\u5317/\u65E5\u672C\u8A9E"}},

		// STATUS -> MailboxStatus
		{`* NOT STATUS`,
			"MailboxStatus", (*MailboxStatus)(nil)},
		{`* STATUS mailbox ()`,
			"MailboxStatus", &MailboxStatus{
				Name: "mailbox"}},
		{`* STATUS inbox (MESSAGES 0)`,
			"MailboxStatus", &MailboxStatus{
				Name: "INBOX"}},
		{`* STATUS "inbox" (MESSAGES 1)`,
			"MailboxStatus", &MailboxStatus{
				Name:     "INBOX",
				Messages: 1}},
		{`* STATUS blurdybloop (MESSAGES 231 UIDNEXT 44292)`,
			"MailboxStatus", &MailboxStatus{
				Name:     "blurdybloop",
				Messages: 231,
				UIDNext:  44292}},
		{`* STATUS *"` + "\u263A!" + `" (MESSAGES 10 RECENT 2 UIDNEXT 42 UIDVALIDITY 123 UNSEEN 5)`,
			"MailboxStatus", &MailboxStatus{
				Name:        "\u263A!",
				Messages:    10,
				Recent:      2,
				UIDNext:     42,
				UIDValidity: 123,
				Unseen:      5}},

		// SEARCH -> []uint32
		{`* NOT SEARCH`,
			"SearchResults", []uint32(nil)},
		{`* SEARCH`,
			"SearchResults", []uint32(nil)},
		{`* SEARCH 1`,
			"SearchResults", []uint32{1}},
		{`* SEARCH 1 2`,
			"SearchResults", []uint32{1, 2}},
		{`* SEARCH 2 3 6`,
			"SearchResults", []uint32{2, 3, 6}},

		// FLAGS and PERMANENTFLAGS -> FlagSet
		{`* NOT FLAGS`,
			"MailboxFlags", FlagSet(nil)},
		{`* FLAGS ()`,
			"MailboxFlags", NewFlagSet()},
		{`* OK [PERMANENTFLAGS ()] No permanent flags permitted`,
			"MailboxFlags", NewFlagSet()},
		{`* FLAGS (\Answered \Flagged \Deleted \Seen \Draft)`,
			"MailboxFlags", NewFlagSet(`\Answered`, `\Flagged`, `\Deleted`, `\Seen`, `\Draft`)},
		{`* OK [PERMANENTFLAGS (\Deleted \Seen \*)] Limited`,
			"MailboxFlags", NewFlagSet(`\Deleted`, `\Seen`, `\*`)},

		// FETCH -> MessageInfo
		{`* 0 NOT FETCH`,
			"MessageInfo", (*MessageInfo)(nil)},
		{`* 1 FETCH ()`,
			"MessageInfo", &MessageInfo{
				Attrs: FieldMap{},
				Seq:   1}},
		{`* 14 FETCH (FLAGS (\Seen \Deleted))`,
			"MessageInfo", &MessageInfo{
				Attrs: FieldMap{"FLAGS": []Field{`\Seen`, `\Deleted`}},
				Seq:   14,
				Flags: NewFlagSet(`\Seen`, `\Deleted`)}},
		{`* 23 FETCH (FLAGS (\Seen) UID 4827313)`,
			"MessageInfo", &MessageInfo{
				Attrs: FieldMap{"FLAGS": []Field{`\Seen`}, "UID": uint32(4827313)},
				Seq:   23,
				UID:   4827313,
				Flags: NewFlagSet(`\Seen`)}},
		{`* 123 FETCH (INTERNALDATE "17-Jul-1996 02:44:25 -0700" RFC822.SIZE 44827)`,
			"MessageInfo", &MessageInfo{
				Attrs:        FieldMap{"INTERNALDATE": `"17-Jul-1996 02:44:25 -0700"`, "RFC822.SIZE": uint32(44827)},
				Seq:          123,
				InternalDate: time.Date(1996, time.July, 17, 2, 44, 25, 0, MST),
				Size:         44827}},
		{`* 4294967295 FETCH (INTERNALDATE " 7-Jul-1996 02:44:25 +0000")`,
			"MessageInfo", &MessageInfo{
				Attrs:        FieldMap{"INTERNALDATE": `" 7-Jul-1996 02:44:25 +0000"`},
				Seq:          4294967295,
				InternalDate: time.Date(1996, time.July, 7, 2, 44, 25, 0, UTC)}},
		{`* 12 FETCH (body[header] {342}` + CRLF + header + ` UID 1 FLAGS () INTERNALDATE "17-Jul-1996 02:44:25 -0700" RFC822.SIZE 1024)`,
			"MessageInfo", &MessageInfo{
				Attrs:        FieldMap{"BODY[HEADER]": lit(header), "UID": uint32(1), "FLAGS": []Field(nil), "INTERNALDATE": `"17-Jul-1996 02:44:25 -0700"`, "RFC822.SIZE": uint32(1024)},
				Seq:          12,
				UID:          1,
				Flags:        NewFlagSet(),
				InternalDate: time.Date(1996, time.July, 17, 2, 44, 25, 0, MST),
				Size:         1024}},

		// QUOTA -> (string, []*Quota)
		{`* NOT QUOTA`,
			"Quota", []interface{}{
				"", []*Quota(nil)}},
		{`* QUOTA "" ()`,
			"Quota", []interface{}{
				"", []*Quota{}}},
		{`* QUOTA "" (STORAGE 10 512)`,
			"Quota", []interface{}{
				"", []*Quota{&Quota{"STORAGE", 10, 512}}}},
		{`* QUOTA "" (STORAGE 10 512 MESSAGE 20 100)`,
			"Quota", []interface{}{
				"", []*Quota{&Quota{"STORAGE", 10, 512}, &Quota{"MESSAGE", 20, 100}}}},
		{`* QUOTA "inbox" (storage 10 512 message 20 100)`,
			"Quota", []interface{}{
				"inbox", []*Quota{&Quota{"STORAGE", 10, 512}, &Quota{"MESSAGE", 20, 100}}}},

		// QUOTAROOT -> (string, []string)
		{`* NOT QUOTAROOT`,
			"QuotaRoot", []interface{}{
				"", []string(nil)}},
		{`* QUOTAROOT comp.mail.mime`,
			"QuotaRoot", []interface{}{
				"comp.mail.mime", []string{}}},
		{`* QUOTAROOT INBOX ""`,
			"QuotaRoot", []interface{}{
				"INBOX", []string{""}}},
		{`* QUOTAROOT "inbox" root1 "root2"`,
			"QuotaRoot", []interface{}{
				"INBOX", []string{"root1", "root2"}}},
	}
	c, s := newTestConn(1024)
	C := newTransport(c, nil)
	r := newReader(C, MemoryReader{}, "A")

	for _, test := range tests {
		C.clear()
		s.Write([]byte(test.in + CRLF))

		raw, err := r.Next()
		rsp, err := raw.Parse()
		if err != nil {
			t.Errorf("Parse(%+q) unexpected error; %v", test.in, err)
			continue
		}

		vout := reflect.ValueOf(rsp).MethodByName(test.call).Call(nil)
		out := make([]interface{}, len(vout))
		for i, v := range vout {
			out[i] = v.Interface()
		}
		if len(out) == 1 {
			if !reflect.DeepEqual(out[0], test.out) {
				t.Errorf("%s(%+q) expected\n%v; got\n%v", test.call, test.in, test.out, out[0])
			}
		} else if !reflect.DeepEqual(out, test.out) {
			t.Errorf("%s(%+q) expected\n%v; got\n%v", test.call, test.in, test.out, out)
		}
	}
}
