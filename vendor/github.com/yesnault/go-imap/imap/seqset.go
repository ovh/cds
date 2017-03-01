// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap

import (
	"fmt"
	"strconv"
	"strings"
)

// SeqSetError is used to report problems with the format of a sequence set
// value.
type SeqSetError string

func (err SeqSetError) Error() string {
	return fmt.Sprintf("imap: bad sequence set value %q", string(err))
}

// seq represents a single seq-number or seq-range value (RFC 3501 ABNF). Values
// may be static (e.g. "1", "2:4") or dynamic (e.g. "*", "1:*"). A seq-number is
// represented by setting start = stop. Zero is used to represent "*", which is
// safe because seq-number uses nz-number rule. The order of values is always
// start <= stop, except when representing "n:*", where start = n and stop = 0.
type seq struct {
	start, stop uint32
}

// parseSeqNumber parses a single seq-number value (non-zero uint32 or "*").
func parseSeqNumber(v string) (uint32, error) {
	if n, err := strconv.ParseUint(v, 10, 32); err == nil && v[0] != '0' {
		return uint32(n), nil
	} else if v == "*" {
		return 0, nil
	}
	return 0, SeqSetError(v)
}

// parseSeq creates a new seq instance by parsing strings in the format "n" or
// "n:m", where n and/or m may be "*". An error is returned for invalid values.
func parseSeq(v string) (s seq, err error) {
	if sep := strings.IndexRune(v, ':'); sep < 0 {
		s.start, err = parseSeqNumber(v)
		s.stop = s.start
		return
	} else if s.start, err = parseSeqNumber(v[:sep]); err == nil {
		if s.stop, err = parseSeqNumber(v[sep+1:]); err == nil {
			if (s.stop < s.start && s.stop != 0) || s.start == 0 {
				s.start, s.stop = s.stop, s.start
			}
			return
		}
	}
	return s, SeqSetError(v)
}

// Contains returns true if the seq-number q is contained in sequence value s.
// The dynamic value "*" contains only other "*" values, the dynamic range "n:*"
// contains "*" and all numbers >= n.
func (s seq) Contains(q uint32) bool {
	if q == 0 {
		return s.stop == 0 // "*" is contained only in "*" and "n:*"
	}
	return s.start != 0 && s.start <= q && (q <= s.stop || s.stop == 0)
}

// Less returns true if s precedes and does not contain seq-number q.
func (s seq) Less(q uint32) bool {
	return (s.stop < q || q == 0) && s.stop != 0
}

// Merge combines sequence values s and t into a single union if the two
// intersect or one is a superset of the other. The order of s and t does not
// matter. If the values cannot be merged, s is returned unmodified and ok is
// set to false.
func (s seq) Merge(t seq) (union seq, ok bool) {
	if union = s; s == t {
		ok = true
		return
	}
	if s.start != 0 && t.start != 0 {
		// s and t are any combination of "n", "n:m", or "n:*"
		if s.start > t.start {
			s, t = t, s
		}
		// s starts at or before t, check where it ends
		if (s.stop >= t.stop && t.stop != 0) || s.stop == 0 {
			return s, true // s is a superset of t
		}
		// s is "n" or "n:m", if m == ^uint32(0) then t is "n:*"
		if s.stop+1 >= t.start || s.stop == ^uint32(0) {
			return seq{s.start, t.stop}, true // s intersects or touches t
		}
		return
	}
	// exactly one of s and t is "*"
	if s.start == 0 {
		if t.stop == 0 {
			return t, true // s is "*", t is "n:*"
		}
	} else if s.stop == 0 {
		return s, true // s is "n:*", t is "*"
	}
	return
}

// String returns sequence value s as a seq-number or seq-range string.
func (s seq) String() string {
	if s.start == s.stop {
		if s.start == 0 {
			return "*"
		}
		return strconv.FormatUint(uint64(s.start), 10)
	}
	b := strconv.AppendUint(make([]byte, 0, 24), uint64(s.start), 10)
	if s.stop == 0 {
		return string(append(b, ':', '*'))
	}
	return string(strconv.AppendUint(append(b, ':'), uint64(s.stop), 10))
}

// SeqSet is used to represent a set of message sequence numbers or UIDs (see
// sequence-set ABNF rule). The zero value is an empty set.
type SeqSet struct {
	set []seq
}

// NewSeqSet returns a new SeqSet instance after parsing the set string.
func NewSeqSet(set string) (s *SeqSet, err error) {
	s = new(SeqSet)
	return s, s.Add(set)
}

// Add inserts new sequence values into the set. The string format is described
// by RFC 3501 sequence-set ABNF rule. If an error is encountered, all values
// inserted successfully prior to the error remain in the set.
func (s *SeqSet) Add(set string) error {
	for _, sv := range strings.Split(set, ",") {
		v, err := parseSeq(sv)
		if err != nil {
			return err
		}
		s.insert(v)
	}
	return nil
}

// AddNum inserts new sequence numbers into the set. The value 0 represents "*".
func (s *SeqSet) AddNum(q ...uint32) {
	for _, v := range q {
		s.insert(seq{v, v})
	}
}

// AddRange inserts a new sequence range into the set.
func (s *SeqSet) AddRange(start, stop uint32) {
	if (stop < start && stop != 0) || start == 0 {
		s.insert(seq{stop, start})
	} else {
		s.insert(seq{start, stop})
	}
}

// AddSet inserts all values from t into s.
func (s *SeqSet) AddSet(t *SeqSet) {
	for _, v := range t.set {
		s.insert(v)
	}
}

// Clear removes all values from the set.
func (s *SeqSet) Clear() {
	s.set = s.set[:0]
}

// Empty returns true if the sequence set does not contain any values.
func (s SeqSet) Empty() bool {
	return len(s.set) == 0
}

// Dynamic returns true if the set contains "*" or "n:*" values.
func (s SeqSet) Dynamic() bool {
	return len(s.set) > 0 && s.set[len(s.set)-1].stop == 0
}

// Contains returns true if the non-zero sequence number or UID q is contained
// in the set. The dynamic range "n:*" contains all q >= n. It is the caller's
// responsibility to handle the special case where q is the maximum UID in the
// mailbox and q < n (i.e. the set cannot match UIDs against "*:n" or "*" since
// it doesn't know what the maximum value is).
func (s SeqSet) Contains(q uint32) bool {
	if _, ok := s.search(q); ok {
		return q != 0
	}
	return false
}

// String returns a sorted representation of all contained sequence values.
func (s SeqSet) String() string {
	if len(s.set) == 0 {
		return ""
	}
	b := make([]byte, 0, 64)
	for _, v := range s.set {
		b = append(b, ',')
		if v.start == 0 {
			b = append(b, '*')
			continue
		}
		b = strconv.AppendUint(b, uint64(v.start), 10)
		if v.start != v.stop {
			if v.stop == 0 {
				b = append(b, ':', '*')
				continue
			}
			b = strconv.AppendUint(append(b, ':'), uint64(v.stop), 10)
		}
	}
	return string(b[1:])
}

// insert adds sequence value v to the set.
func (s *SeqSet) insert(v seq) {
	i, _ := s.search(v.start)
	merged := false
	if i > 0 {
		// try merging with the preceding entry (e.g. "1,4".insert(2), i == 1)
		s.set[i-1], merged = s.set[i-1].Merge(v)
	}
	if i == len(s.set) {
		// v was either merged with the last entry or needs to be appended
		if !merged {
			s.insertAt(i, v)
		}
		return
	} else if merged {
		i--
	} else if s.set[i], merged = s.set[i].Merge(v); !merged {
		s.insertAt(i, v) // insert in the middle (e.g. "1,5".insert(3), i == 1)
		return
	}
	// v was merged with s.set[i], continue trying to merge until the end
	for j := i + 1; j < len(s.set); j++ {
		if s.set[i], merged = s.set[i].Merge(s.set[j]); !merged {
			if j > i+1 {
				// cut out all entries between i and j that were merged
				s.set = append(s.set[:i+1], s.set[j:]...)
			}
			return
		}
	}
	// everything after s.set[i] was merged
	s.set = s.set[:i+1]
}

// insertAt inserts a new sequence value v at index i, resizing s.set as needed.
func (s *SeqSet) insertAt(i int, v seq) {
	if n := len(s.set); i == n {
		// insert at the end
		s.set = append(s.set, v)
		return
	} else if n < cap(s.set) {
		// enough space, shift everything at and after i to the right
		s.set = s.set[:n+1]
		copy(s.set[i+1:], s.set[i:])
	} else {
		// allocate new slice and copy everything, n is at least 1
		set := make([]seq, n+1, n*2)
		copy(set, s.set[:i])
		copy(set[i+1:], s.set[i:])
		s.set = set
	}
	s.set[i] = v
	return
}

// search attempts to find the index of the sequence set value that contains q.
// If no values contain q, the returned index is the position where q should be
// inserted and ok is set to false.
func (s SeqSet) search(q uint32) (i int, ok bool) {
	min, max := 0, len(s.set)-1
	for min < max {
		if mid := (min + max) >> 1; s.set[mid].Less(q) {
			min = mid + 1
		} else {
			max = mid
		}
	}
	if max < 0 || s.set[min].Less(q) {
		return len(s.set), false // q is the new largest value
	}
	return min, s.set[min].Contains(q)
}
