// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import "strconv"

// ConnState represents client connection states. See RFC 3501 page 15 for a
// state diagram.
type ConnState uint8

// Client connection states.
const (
	unknown  = ConnState(1 << iota) // Pre-greeting internal state
	Login                           // Not authenticated
	Auth                            // Authenticated
	Selected                        // Mailbox selected
	Logout                          // Connection closing
	Closed   = ConnState(0)         // Connection closed
)

var connStates = []enumName{
	{uint32(unknown), "unknown"},
	{uint32(Login), "Login"},
	{uint32(Auth), "Auth"},
	{uint32(Selected), "Selected"},
	{uint32(Logout), "Logout"},
	{uint32(Closed), "Closed"},
}

func (v ConnState) String() string   { return enumString(uint32(v), connStates, false) }
func (v ConnState) GoString() string { return enumString(uint32(v), connStates, true) }

// RespType indicates the type of information contained in the response.
type RespType uint8

// Server response types.
const (
	Status   = RespType(1 << iota) // Untagged status
	Data                           // Untagged data
	Continue                       // Continuation request
	Done                           // Tagged command completion
)

var respTypes = []enumName{
	{uint32(Status), "Status"},
	{uint32(Data), "Data"},
	{uint32(Continue), "Continue"},
	{uint32(Done), "Done"},
}

func (v RespType) String() string   { return enumString(uint32(v), respTypes, false) }
func (v RespType) GoString() string { return enumString(uint32(v), respTypes, true) }

// RespStatus is the code sent in status messages to indicate success, failure,
// or changes in the connection state.
type RespStatus uint8

// Status conditions used by Status and Done response types.
const (
	OK      = RespStatus(1 << iota) // Success
	NO                              // Operational error
	BAD                             // Protocol-level error
	PREAUTH                         // Greeting status indicating Auth state (untagged-only)
	BYE                             // Connection closing (untagged-only)
)

var respStatuses = []enumName{
	{uint32(OK), "OK"},
	{uint32(NO), "NO"},
	{uint32(BAD), "BAD"},
	{uint32(PREAUTH), "PREAUTH"},
	{uint32(BYE), "BYE"},
}

func (v RespStatus) String() string   { return enumString(uint32(v), respStatuses, false) }
func (v RespStatus) GoString() string { return enumString(uint32(v), respStatuses, true) }

// FieldType describes the data type of a single response field.
type FieldType uint8

// Valid field data types.
const (
	Atom          = FieldType(1 << iota) // String consisting of non-special ASCII characters
	Number                               // Unsigned 32-bit integer
	QuotedString                         // String enclosed in double quotes
	LiteralString                        // String or binary data
	List                                 // Parenthesized list
	Bytes                                // Decoded Base64 data
	NIL                                  // Case-insensitive atom string "NIL"
)

var fieldTypes = []enumName{
	{uint32(Atom), "Atom"},
	{uint32(Number), "Number"},
	{uint32(QuotedString), "QuotedString"},
	{uint32(LiteralString), "LiteralString"},
	{uint32(List), "List"},
	{uint32(Bytes), "Bytes"},
	{uint32(NIL), "NIL"},
}

func (v FieldType) String() string   { return enumString(uint32(v), fieldTypes, false) }
func (v FieldType) GoString() string { return enumString(uint32(v), fieldTypes, true) }

// LogMask represents the categories of debug messages that can be logged by the
// Client.
type LogMask uint8

// Debug message categories.
const (
	LogConn  = LogMask(1 << iota)   // Connection events
	LogState                        // State changes
	LogCmd                          // Command execution
	LogRaw                          // Raw data stream excluding literals
	LogGo                           // Goroutine execution
	LogAll   = LogMask(1<<iota - 1) // All messages
	LogNone  = LogMask(0)           // No messages
)

var logMasks = []enumName{
	{uint32(LogAll), "LogAll"},
	{uint32(LogConn), "LogConn"},
	{uint32(LogState), "LogState"},
	{uint32(LogCmd), "LogCmd"},
	{uint32(LogRaw), "LogRaw"},
	{uint32(LogNone), "LogNone"},
}

func (v LogMask) String() string   { return enumString(uint32(v), logMasks, false) }
func (v LogMask) GoString() string { return enumString(uint32(v), logMasks, true) }

// enumName associates an enum value with its name for printing.
type enumName struct {
	v uint32
	s string
}

// enumString converts a flag-based enum value into its string representation.
func enumString(v uint32, names []enumName, goSyntax bool) string {
	s := ""
	for _, n := range names {
		if v&n.v == n.v && (n.v != 0 || v == 0) {
			if len(s) > 0 {
				s += "+"
			}
			if goSyntax {
				s += "imap."
			}
			s += n.s
			if v &= ^n.v; v == 0 {
				return s
			}
		}
	}
	if len(s) > 0 {
		s += "+"
	}
	return s + "0x" + strconv.FormatUint(uint64(v), 16)
}
