// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Date-time format used by INTERNALDATE.
const DATETIME = `"_2-Jan-2006 15:04:05 -0700"`

// Field represents a single data item in a command or response. Fields are
// separated from one another by a single space. Field slices represent
// parenthesized lists.
type Field interface{}

// TypeOf returns the field data type. Valid types are Atom, Number,
// QuotedString, LiteralString, List, Bytes, and NIL. Zero is returned for
// unknown data types.
func TypeOf(f Field) FieldType {
	switch f.(type) {
	case string:
		if Quoted(f) {
			return QuotedString
		} else if len(f.(string)) > 0 {
			return Atom
		}
	case uint32:
		return Number
	case []Field:
		return List
	case []byte:
		return Bytes
	case Literal:
		return LiteralString
	case nil:
		return NIL
	}
	return 0
}

// AsAtom returns the value of an atom field. An empty string is returned if
// TypeOf(f) != Atom.
func AsAtom(f Field) string {
	if v, ok := f.(string); ok && !Quoted(f) {
		return v
	}
	return ""
}

// AsNumber returns the value of a numeric field. Zero is returned if TypeOf(f)
// != Number.
func AsNumber(f Field) uint32 {
	v, _ := f.(uint32)
	return v
}

// AsString returns the value of an astring (string or atom) field. Quoted
// strings are decoded to their original representation. An empty string is
// returned if TypeOf(f)&(Atom|QuotedString|LiteralString) == 0 or the string is
// invalid.
func AsString(f Field) string {
	if v, ok := f.(string); ok {
		if Quoted(f) {
			v, _ = Unquote(v)
		}
		return v
	} else if _, ok = f.(Literal); ok {
		return string(AsBytes(f))
	}
	return ""
}

// AsBytes returns the value of a data field. Nil is returned if
// TypeOf(f)&(QuotedString|LiteralString|Bytes) == 0.
func AsBytes(f Field) []byte {
	switch v := f.(type) {
	case []byte:
		return v
	case string:
		if Quoted(f) {
			b, _ := unquote([]byte(v))
			return b
		}
	case *literal:
		return v.data
	case Literal:
		if n := v.Info().Len; n > 0 {
			b := bytes.NewBuffer(make([]byte, 0, n))
			if _, err := v.WriteTo(b); err == nil && uint32(b.Len()) == n {
				return b.Bytes()
			}
		}
	}
	return nil
}

// AsList returns the value of a parenthesized list. Nil is returned if
// TypeOf(f) != List.
func AsList(f Field) []Field {
	v, _ := f.([]Field)
	return v
}

// AsDateTime returns the value of a date-time quoted string field (e.g.
// INTERNALDATE). The zero value of time.Time is returned if f does not contain
// a valid date-time string.
func AsDateTime(f Field) time.Time {
	s, _ := f.(string)
	if v, err := time.Parse(DATETIME, s); err == nil {
		return v
	}
	return time.Time{}
}

// AsMailbox returns the value of a mailbox name field. All valid atoms and
// strings encoded as quoted UTF-8 or modified UTF-7 are decoded appropriately.
// The special case-insensitive name "INBOX" is always converted to upper case.
func AsMailbox(f Field) string {
	v := AsString(f)
	if len(v) == 5 && toUpper(v) == "INBOX" {
		return "INBOX"
	} else if !QuotedUTF8(f) {
		if s, err := UTF7Decode(v); err == nil {
			return s
		}
	}
	return v
}

// FieldMap represents key-value pairs of data items, such as those returned in
// a FETCH response. Key names are atoms converted to upper case.
type FieldMap map[string]Field

// AsFieldMap returns a map of key-value pairs extracted from a parenthesized
// list. Nil is returned if TypeOf(f) != List, the number of fields in the list
// is not even, or one of the keys is not an Atom.
func AsFieldMap(f Field) FieldMap {
	list, ok := f.([]Field)
	n := len(list)
	if !ok || n&1 == 1 { // n must be even; initialize the map for n == 0
		return nil
	}
	v := make(FieldMap, n/2)
	for i := 0; i < n; i += 2 {
		if k := toUpper(AsAtom(list[i])); k != "" {
			v[k] = list[i+1]
		} else {
			return nil
		}
	}
	return v
}

func (fm FieldMap) String() string {
	if len(fm) == 0 {
		return "()"
	}
	v, i := make([]string, len(fm)), 0
	for k := range fm {
		v[i] = k
		i++
	}
	sort.Strings(v)
	for i, k := range v {
		v[i] = fmt.Sprint(k, ":", fm[k])
	}
	return "(" + strings.Join(v, " ") + ")"
}

// FlagSet represents the flags enabled for a single mailbox or message. The map
// values are always set to true; a flag must be deleted from the map to
// indicate that it is not enabled.
type FlagSet map[string]bool

// NewFlagSet returns a new flag set with the specified flags enabled.
func NewFlagSet(flags ...string) FlagSet {
	fs := make(FlagSet, len(flags))
	for _, v := range flags {
		fs[v] = true
	}
	return fs
}

// AsFlags returns a set of flags extracted from a parenthesized list. The
// function does not check every atom for the leading backslash, because it is
// not permitted in user-defined flags (keywords). Nil is returned if TypeOf(f)
// != List or one of the fields is not an atom.
func AsFlagSet(f Field) FlagSet {
	list, ok := f.([]Field)
	if !ok {
		return nil
	}
	v := make(FlagSet, len(list))
	for _, f := range list {
		if s := AsAtom(f); s != "" {
			v[s] = true
		} else {
			return nil
		}
	}
	return v
}

// Replace removes all existing flags from the set and inserts new ones.
func (fs FlagSet) Replace(f Field) {
	if list, ok := f.([]Field); ok {
		for v := range fs {
			delete(fs, v)
		}
		for _, f := range list {
			if v := AsAtom(f); v != "" {
				fs[v] = true
			}
		}
	}
}

func (fs FlagSet) String() string {
	if len(fs) == 0 {
		return "()"
	}
	v, i := make([]string, len(fs)), 0
	for k := range fs {
		v[i] = k
		i++
	}
	sort.Strings(v)
	return "(" + strings.Join(v, " ") + ")"
}

// intValue converts any signed integer value to int64. It panics if f is not a
// signed integer.
func intValue(f Field) int64 {
	switch v := f.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return int64(v)
	}
	panic("imap: not an int")
}

// uintValue converts any unsigned integer value to uint64. It panics if f is
// not an unsigned integer.
func uintValue(f Field) uint64 {
	switch v := f.(type) {
	case uint:
		return uint64(v)
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return uint64(v)
	}
	panic("imap: not a uint")
}
