// Copyright 2015 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package stack

import (
	"sort"
)

// Similarity is the level at which two call lines arguments must match to be
// considered similar enough to coalesce them.
type Similarity int

const (
	// ExactFlags requires same bits (e.g. Locked).
	ExactFlags Similarity = iota
	// ExactLines requests the exact same arguments on the call line.
	ExactLines
	// AnyPointer considers different pointers a similar call line.
	AnyPointer
	// AnyValue accepts any value as similar call line.
	AnyValue
)

// Aggregate merges similar goroutines into buckets.
//
// The buckets are ordered in library provided order of relevancy. You can
// reorder at your chosing.
func Aggregate(goroutines []*Goroutine, similar Similarity) []*Bucket {
	type count struct {
		ids   []int
		first bool
	}
	b := map[*Signature]*count{}
	// O(n²). Fix eventually.
	for _, routine := range goroutines {
		found := false
		for key, c := range b {
			// When a match is found, this effectively drops the other goroutine ID.
			if key.similar(&routine.Signature, similar) {
				found = true
				c.ids = append(c.ids, routine.ID)
				c.first = c.first || routine.First
				if !key.equal(&routine.Signature) {
					// Almost but not quite equal. There's different pointers passed
					// around but the same values. Zap out the different values.
					newKey := key.merge(&routine.Signature)
					b[newKey] = c
					delete(b, key)
				}
				break
			}
		}
		if !found {
			// Create a copy of the Signature, since it will be mutated.
			key := &Signature{}
			*key = routine.Signature
			b[key] = &count{ids: []int{routine.ID}, first: routine.First}
		}
	}
	out := make(buckets, 0, len(b))
	for signature, c := range b {
		sort.Ints(c.ids)
		out = append(out, &Bucket{Signature: *signature, IDs: c.ids, First: c.first})
	}
	sort.Sort(out)
	return out
}

// Bucket is a stack trace signature and the list of goroutines that fits this
// signature.
type Bucket struct {
	Signature
	// IDs is the ID of each Goroutine with this Signature.
	IDs []int
	// First is true if this Bucket contains the first goroutine, e.g. the one
	// Signature that likely generated the panic() call, if any.
	First bool
}

// less does reverse sort.
func (b *Bucket) less(r *Bucket) bool {
	if b.First || r.First {
		return b.First
	}
	return b.Signature.less(&r.Signature)
}

//

// buckets is a list of Bucket sorted by repeation count.
type buckets []*Bucket

func (b buckets) Len() int {
	return len(b)
}

func (b buckets) Less(i, j int) bool {
	return b[i].less(b[j])
}

func (b buckets) Swap(i, j int) {
	b[j], b[i] = b[i], b[j]
}
