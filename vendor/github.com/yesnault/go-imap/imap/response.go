// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"fmt"
	"time"
)

// Response represents a single status, data, or command continuation response.
// All response types are parsed into the same general format, which can then be
// decoded to more specific representations either by calling the provided
// decoder methods, or by manually navigating Fields and other attributes. Here
// are a few examples of the parser output:
//
// 	S: * CAPABILITY IMAP4rev1 STARTTLS AUTH=GSSAPI
// 	S: * OK [UNSEEN 12] Message 12 is first unseen
// 	S: A142 OK [read-write] SELECT completed
//
// 	Response objects:
//
// 	&imap.Response{
// 		Raw:    []byte("* CAPABILITY IMAP4rev1 STARTTLS AUTH=GSSAPI"),
// 		Tag:    "*",
// 		Type:   imap.Data,
// 		Label:  "CAPABILITY",
// 		Fields: []Field{"CAPABILITY", "IMAP4rev1", "STARTTLS", "AUTH=GSSAPI"},
// 	}
// 	&imap.Response{
// 		Raw:    []byte("* OK [UNSEEN 12] Message 12 is first unseen"),
// 		Tag:    "*",
// 		Type:   imap.Status,
// 		Status: imap.OK,
// 		Info:   "Message 12 is first unseen",
// 		Label:  "UNSEEN",
// 		Fields: []Field{"UNSEEN", uint32(12)},
// 	}
// 	&imap.Response{
// 		Raw:    []byte("A142 OK [read-write] SELECT completed"),
// 		Tag:    "A142",
// 		Type:   imap.Done,
// 		Status: imap.OK,
// 		Info:   "SELECT completed",
// 		Label:  "READ-WRITE",
// 		Fields: []Field{"read-write"},
// 	}
type Response struct {
	// Order in which this response was received, starting at 1 for the server
	// greeting.
	Order int64

	// Original response line from which this Response object was constructed.
	// Literal strings and CRLFs are omitted.
	Raw []byte

	// All literal strings in the order they were received. Do not assume that a
	// FETCH request for BODY[], for example, will return exactly one literal
	// with the requested data. Use the decoder methods, or navigate Fields
	// according to the response format, to get the desired information.
	Literals []Literal

	// Response tag ("*", "+", or command tag).
	Tag string

	// Response type (Status, Data, Continue, or Done).
	Type RespType

	// Status condition if Type is Status or Done (OK, NO, BAD, PREAUTH, or
	// BYE). Only OK, NO, and BAD may be used in tagged (Done) responses.
	Status RespStatus

	// Human-readable text portion of a Status, Continue, or Done response, or
	// the original Base64 text of a challenge-response authentication request.
	Info string

	// First atom in Fields (usually index 0 or 1) converted to upper case. This
	// determines the format of Fields, as described in RFC 3501 section 7. A
	// Continue response containing Base64 data is labeled "BASE64".
	Label string

	// Data or response code fields extracted by the parser. For a Data
	// response, this is everything after the "*" tag. For a Status or Done
	// response, this is the response code (if there is one). For a Continue
	// response with a "BASE64" Label, Fields[0] is the decoded byte slice.
	Fields []Field

	// Cached decoder output. This is used by the decoder methods to avoid
	// traversing Fields multiple times. User code should not modify or access
	// this field except when writing a custom decoder (see response.go for
	// examples).
	Decoded interface{}
}

// String returns the raw text from which this Response object was constructed.
// Literal strings and CRLFs are omitted.
func (rsp *Response) String() string {
	return string(rsp.Raw)
}

// Value returns the first unsigned 32-bit integer in Fields without descending
// into parenthesized lists. This decoder is primarily intended for Status/Data
// responses labeled EXISTS, RECENT, EXPUNGE, UNSEEN, UIDNEXT, and UIDVALIDITY.
func (rsp *Response) Value() uint32 {
	v, ok := rsp.Decoded.(uint32)
	if !ok && rsp.Decoded == nil {
		for _, f := range rsp.Fields {
			if TypeOf(f) == Number {
				v = AsNumber(f)
				rsp.Decoded = v
				break
			}
		}
	}
	return v
}

// Challenge returns the decoded Base64 data from a continuation request sent
// during challenge-response authentication.
func (rsp *Response) Challenge() []byte {
	v, ok := rsp.Decoded.([]byte)
	if !ok && rsp.Decoded == nil && rsp.Label == "BASE64" {
		v = AsBytes(rsp.Fields[0])
		rsp.Decoded = v
	}
	return v
}

// MailboxInfo represents the mailbox attributes returned in a LIST or LSUB
// response.
type MailboxInfo struct {
	Attrs FlagSet // Mailbox attributes (e.g. `\Noinferiors`, `\Noselect`)
	Delim string  // Hierarchy delimiter (empty string == NIL, i.e. flat name)
	Name  string  // Mailbox name decoded to UTF-8
}

// MailboxInfo returns the mailbox attributes extracted from a LIST or LSUB
// response.
func (rsp *Response) MailboxInfo() *MailboxInfo {
	v, ok := rsp.Decoded.(*MailboxInfo)
	if !ok && rsp.Decoded == nil &&
		(rsp.Label == "LIST" || rsp.Label == "LSUB") {
		v = &MailboxInfo{
			Attrs: AsFlagSet(rsp.Fields[1]),
			Delim: AsString(rsp.Fields[2]),
			Name:  AsMailbox(rsp.Fields[3]),
		}
		rsp.Decoded = v
	}
	return v
}

// MailboxStatus represents the mailbox status information returned in a STATUS
// response. It is also used by the Client to keep an updated view of the
// currently selected mailbox. Fields that are only set by the Client are marked
// as client-only.
type MailboxStatus struct {
	Name         string  // Mailbox name
	ReadOnly     bool    // Mailbox read/write access (client-only)
	Flags        FlagSet // Defined flags in the mailbox (client-only)
	PermFlags    FlagSet // Flags that the client can change permanently (client-only)
	Messages     uint32  // Number of messages in the mailbox
	Recent       uint32  // Number of messages with the \Recent flag set
	Unseen       uint32  // Sequence number of the first unseen message
	UIDNext      uint32  // The next unique identifier value
	UIDValidity  uint32  // The unique identifier validity value
	UIDNotSticky bool    // UIDPLUS extension (client-only)
}

// newMailboxStatus returns an initialized MailboxStatus instance.
func newMailboxStatus(name string) *MailboxStatus {
	if len(name) == 5 && toUpper(name) == "INBOX" {
		name = "INBOX"
	}
	return &MailboxStatus{
		Name:      name,
		Flags:     make(FlagSet),
		PermFlags: make(FlagSet),
	}
}

func (m *MailboxStatus) String() string {
	return fmt.Sprintf("--- %+q ---\n"+
		"ReadOnly:     %v\n"+
		"Flags:        %v\n"+
		"PermFlags:    %v\n"+
		"Messages:     %v\n"+
		"Recent:       %v\n"+
		"Unseen:       %v\n"+
		"UIDNext:      %v\n"+
		"UIDValidity:  %v\n"+
		"UIDNotSticky: %v\n",
		m.Name, m.ReadOnly, m.Flags, m.PermFlags, m.Messages, m.Recent,
		m.Unseen, m.UIDNext, m.UIDValidity, m.UIDNotSticky)
}

// MailboxStatus returns the mailbox status information extracted from a STATUS
// response.
func (rsp *Response) MailboxStatus() *MailboxStatus {
	v, ok := rsp.Decoded.(*MailboxStatus)
	if !ok && rsp.Decoded == nil && rsp.Label == "STATUS" {
		v = &MailboxStatus{Name: AsMailbox(rsp.Fields[1])}
		f := AsList(rsp.Fields[2])
		for i := 0; i < len(f)-1; i += 2 {
			switch n := AsNumber(f[i+1]); toUpper(AsAtom(f[i])) {
			case "MESSAGES":
				v.Messages = n
			case "RECENT":
				v.Recent = n
			case "UIDNEXT":
				v.UIDNext = n
			case "UIDVALIDITY":
				v.UIDValidity = n
			case "UNSEEN":
				v.Unseen = n
			}
		}
		rsp.Decoded = v
	}
	return v
}

// SearchResults returns a slice of message sequence numbers or UIDs extracted
// from a SEARCH response.
func (rsp *Response) SearchResults() []uint32 {
	v, ok := rsp.Decoded.([]uint32)
	if !ok && rsp.Decoded == nil && rsp.Label == "SEARCH" {
		if len(rsp.Fields) > 1 {
			v = make([]uint32, len(rsp.Fields)-1)
			for i, f := range rsp.Fields[1:] {
				v[i] = AsNumber(f)
			}
		}
		rsp.Decoded = v
	}
	return v
}

// MailboxFlags returns a FlagSet extracted from a FLAGS or PERMANENTFLAGS
// response. Note that FLAGS is a Data response, while PERMANENTFLAGS is Status.
func (rsp *Response) MailboxFlags() FlagSet {
	v, ok := rsp.Decoded.(FlagSet)
	if !ok && rsp.Decoded == nil &&
		(rsp.Label == "FLAGS" || rsp.Label == "PERMANENTFLAGS") {
		v = AsFlagSet(rsp.Fields[1])
		rsp.Decoded = v
	}
	return v
}

// MessageInfo represents the message attributes returned in a FETCH response.
// The values of attributes marked optional are valid only if that attribute
// also appears in Attrs (e.g. UID is valid if and only if Attrs["UID"] != nil).
// These attributes are extracted from Attrs purely for convenience.
type MessageInfo struct {
	Attrs        FieldMap  // All returned attributes
	Seq          uint32    // Message sequence number
	UID          uint32    // Unique identifier (optional in non-UID FETCH)
	Flags        FlagSet   // Flags that are set for this message (optional)
	InternalDate time.Time // Internal to the server message timestamp (optional)
	Size         uint32    // Message size in bytes (optional)
}

// MessageInfo returns the message attributes extracted from a FETCH response.
func (rsp *Response) MessageInfo() *MessageInfo {
	v, ok := rsp.Decoded.(*MessageInfo)
	if !ok && rsp.Decoded == nil && rsp.Label == "FETCH" {
		kv := AsFieldMap(rsp.Fields[2])
		v = &MessageInfo{
			Attrs:        kv,
			Seq:          AsNumber(rsp.Fields[0]),
			UID:          AsNumber(kv["UID"]),
			Flags:        AsFlagSet(kv["FLAGS"]),
			InternalDate: AsDateTime(kv["INTERNALDATE"]),
			Size:         AsNumber(kv["RFC822.SIZE"]),
		}
		rsp.Decoded = v
	}
	return v
}

// Quota represents a single resource limit on a mailbox quota root returned in
// a QUOTA response, as described in RFC 2087.
type Quota struct {
	Resource string // Resource name (e.g. STORAGE, MESSAGE)
	Usage    uint32 // Current usage (in units of 1024 octets for STORAGE)
	Limit    uint32 // Current limit
}

// Quota returns the resource quotas extracted from a QUOTA response.
func (rsp *Response) Quota() (root string, quota []*Quota) {
	type vt struct {
		root  string
		quota []*Quota
	}
	v, ok := rsp.Decoded.(*vt)
	if !ok && rsp.Decoded == nil && rsp.Label == "QUOTA" {
		list := AsList(rsp.Fields[2])
		if len(list)%3 != 0 {
			return
		}
		root = AsString(rsp.Fields[1])
		quota = make([]*Quota, len(list)/3)
		for i := 0; i < len(list); i += 3 {
			quota[i/3] = &Quota{
				Resource: toUpper(AsAtom(list[i])),
				Usage:    AsNumber(list[i+1]),
				Limit:    AsNumber(list[i+2]),
			}
		}
		rsp.Decoded = &vt{root, quota}
	} else if ok {
		root, quota = v.root, v.quota
	}
	return
}

// QuotaRoot returns the mailbox name and associated quota roots from a
// QUOTAROOT response.
func (rsp *Response) QuotaRoot() (mbox string, roots []string) {
	type vt struct {
		mbox  string
		roots []string
	}
	v, ok := rsp.Decoded.(*vt)
	if !ok && rsp.Decoded == nil && rsp.Label == "QUOTAROOT" {
		mbox = AsMailbox(rsp.Fields[1])
		roots = make([]string, len(rsp.Fields[2:]))
		for i, root := range rsp.Fields[2:] {
			roots[i] = AsString(root)
		}
		rsp.Decoded = &vt{mbox, roots}
	} else if ok {
		mbox, roots = v.mbox, v.roots
	}
	return
}

// ResponseError wraps a Response pointer for use in an error context, such as
// when a command fails with a NO or BAD status condition. For Status and Done
// response types, the value of Response.Info may be presented to the user.
// Reason provides additional information about the cause of the error.
type ResponseError struct {
	*Response
	Reason string
}

func (rsp ResponseError) Error() string {
	line, ellipsis := rsp.Raw, ""
	if len(line) > rawLimit {
		line, ellipsis = line[:rawLimit], "..."
	}
	return fmt.Sprintf("imap: %s (%+q%s)", rsp.Reason, line, ellipsis)
}
