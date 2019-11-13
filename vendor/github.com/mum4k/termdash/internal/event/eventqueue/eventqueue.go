// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package eventqueue provides an unboud FIFO queue of events.
package eventqueue

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/mum4k/termdash/terminal/terminalapi"
)

// node is a single data item on the queue.
type node struct {
	prev  *node
	next  *node
	event terminalapi.Event
}

// Unbound is an unbound FIFO queue of terminal events.
// Unbound must not be copied, pass it by reference only.
// This implementation is thread-safe.
type Unbound struct {
	first *node
	last  *node
	// mu protects first and last.
	mu sync.Mutex

	// cond is used to notify any callers waiting on a call to Pull().
	cond *sync.Cond

	// condMU protects cond.
	condMU sync.RWMutex

	// done is closed when the queue isn't needed anymore.
	done chan struct{}
}

// New returns a new Unbound queue of terminal events.
// Call Close() when done with the queue.
func New() *Unbound {
	u := &Unbound{
		done: make(chan (struct{})),
	}
	u.cond = sync.NewCond(&u.condMU)
	go u.wake() // Stops when Close() is called.
	return u
}

// wake periodically wakes up all goroutines waiting at Pull() so they can
// check if the context expired.
func (u *Unbound) wake() {
	const spinTime = 250 * time.Millisecond
	t := time.NewTicker(spinTime)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			u.cond.Broadcast()
		case <-u.done:
			return
		}
	}
}

// Empty determines if the queue is empty.
func (u *Unbound) Empty() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.empty()
}

// empty determines if the queue is empty.
func (u *Unbound) empty() bool {
	return u.first == nil
}

// Push pushes an event onto the queue.
func (u *Unbound) Push(e terminalapi.Event) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.push(e)
}

// push is the implementation of Push.
// Caller must hold u.mu.
func (u *Unbound) push(e terminalapi.Event) {
	n := &node{
		event: e,
	}
	if u.empty() {
		u.first = n
		u.last = n
	} else {
		prev := u.last
		u.last.next = n
		u.last = n
		u.last.prev = prev
	}
	u.cond.Signal()
}

// Pop pops an event from the queue. Returns nil if the queue is empty.
func (u *Unbound) Pop() terminalapi.Event {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.empty() {
		return nil
	}

	n := u.first
	u.first = u.first.next

	if u.empty() {
		u.last = nil
	}
	return n.event
}

// Pull is like Pop(), but blocks until an item is available or the context
// expires. Returns a nil event if the context expired.
func (u *Unbound) Pull(ctx context.Context) terminalapi.Event {
	if e := u.Pop(); e != nil {
		return e
	}

	u.cond.L.Lock()
	defer u.cond.L.Unlock()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if e := u.Pop(); e != nil {
			return e
		}
		u.cond.Wait()
	}
}

// Close should be called when the queue isn't needed anymore.
func (u *Unbound) Close() {
	close(u.done)
}

// Throttled is an unbound and throttled FIFO queue of terminal events.
// Throttled must not be copied, pass it by reference only.
// This implementation is thread-safe.
type Throttled struct {
	queue *Unbound
	max   int
}

// NewThrottled returns a new Throttled queue of terminal events.
//
// This queue scans the queue content on each Push call and won't Push the
// event if there already is a continuous chain of exactly the same events
// en queued. The argument maxRep specifies the maximum number of repetitive
// events.
//
// Call Close() when done with the queue.
func NewThrottled(maxRep int) *Throttled {
	t := &Throttled{
		queue: New(),
		max:   maxRep,
	}
	return t
}

// Empty determines if the queue is empty.
func (t *Throttled) Empty() bool {
	return t.queue.empty()
}

// Push pushes an event onto the queue.
func (t *Throttled) Push(e terminalapi.Event) {
	t.queue.mu.Lock()
	defer t.queue.mu.Unlock()

	if t.queue.empty() {
		t.queue.push(e)
		return
	}

	var same int
	for n := t.queue.last; n != nil; n = n.prev {
		if reflect.DeepEqual(e, n.event) {
			same++
		} else {
			break
		}

		if same > t.max {
			return // Drop the repetitive event.
		}
	}
	t.queue.push(e)
}

// Pop pops an event from the queue. Returns nil if the queue is empty.
func (t *Throttled) Pop() terminalapi.Event {
	return t.queue.Pop()
}

// Pull is like Pop(), but blocks until an item is available or the context
// expires. Returns a nil event if the context expired.
func (t *Throttled) Pull(ctx context.Context) terminalapi.Event {
	return t.queue.Pull(ctx)
}

// Close should be called when the queue isn't needed anymore.
func (t *Throttled) Close() {
	close(t.queue.done)
}
