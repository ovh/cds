// Copyright 2019 Google Inc.
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

// Package event provides a non-blocking event distribution and subscription
// system.
package event

import (
	"context"
	"reflect"
	"sync"

	"github.com/mum4k/termdash/internal/event/eventqueue"
	"github.com/mum4k/termdash/terminal/terminalapi"
)

// Callback is a function provided by an event subscriber.
// It gets called with each event that passed the subscription filter.
// Implementations must be thread-safe, events come from a separate goroutine.
// Implementation should be light-weight, otherwise a slow-processing
// subscriber can build a long tail of events.
type Callback func(terminalapi.Event)

// queue is a queue of terminal events.
type queue interface {
	Push(e terminalapi.Event)
	Pull(ctx context.Context) terminalapi.Event
	Close()
}

// subscriber represents a single subscriber.
type subscriber struct {
	// cb is the callback the subscriber receives events on.
	cb Callback

	// filter filters events towards the subscriber.
	// An empty filter receives all events.
	filter map[reflect.Type]bool

	// queue is a queue of events towards the subscriber.
	queue queue

	// cancel when called terminates the goroutine that forwards events towards
	// this subscriber.
	cancel context.CancelFunc

	// processes is the number of events that were fully processed, i.e.
	// delivered to the callback.
	processed int

	// mu protects busy.
	mu sync.Mutex
}

// newSubscriber creates a new event subscriber.
func newSubscriber(filter []terminalapi.Event, cb Callback, opts *subscribeOptions) *subscriber {
	f := map[reflect.Type]bool{}
	for _, ev := range filter {
		f[reflect.TypeOf(ev)] = true
	}

	ctx, cancel := context.WithCancel(context.Background())
	var q queue
	if opts.throttle {
		q = eventqueue.NewThrottled(opts.maxRep)
	} else {
		q = eventqueue.New()
	}

	s := &subscriber{
		cb:     cb,
		filter: f,
		queue:  q,
		cancel: cancel,
	}

	// Terminates when stop() is called.
	go s.run(ctx)
	return s
}

// callback sends the event to the callback.
func (s *subscriber) callback(ev terminalapi.Event) {
	s.cb(ev)

	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.processed++
	}()
}

// run periodically forwards events towards the subscriber.
// Terminates when the context expires.
func (s *subscriber) run(ctx context.Context) {
	for {
		ev := s.queue.Pull(ctx)
		if ev != nil {
			s.callback(ev)
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// event forwards an event to the subscriber.
func (s *subscriber) event(ev terminalapi.Event) {
	if len(s.filter) == 0 {
		s.queue.Push(ev)
	}

	t := reflect.TypeOf(ev)
	if s.filter[t] {
		s.queue.Push(ev)
	}
}

// processedEvents returns the number of events processed by this subscriber.
func (s *subscriber) processedEvents() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.processed
}

// stop stops the event subscriber.
func (s *subscriber) stop() {
	s.cancel()
	s.queue.Close()
}

// DistributionSystem distributes events to subscribers.
//
// Subscribers can request filtering of events they get based on event type or
// subscribe to all events.
//
// The distribution system maintains a queue towards each subscriber, making
// sure that a single slow subscriber only slows itself down, rather than the
// entire application.
//
// This object is thread-safe.
type DistributionSystem struct {
	// subscribers subscribe to events.
	// maps subscriber id to subscriber.
	subscribers map[int]*subscriber

	// nextID is id for the next subscriber.
	nextID int

	// mu protects the distribution system.
	mu sync.Mutex
}

// NewDistributionSystem creates a new event distribution system.
func NewDistributionSystem() *DistributionSystem {
	return &DistributionSystem{
		subscribers: map[int]*subscriber{},
	}
}

// Event should be called with events coming from the terminal.
// The distribution system will distribute these to all the subscribers.
func (eds *DistributionSystem) Event(ev terminalapi.Event) {
	eds.mu.Lock()
	defer eds.mu.Unlock()

	for _, sub := range eds.subscribers {
		sub.event(ev)
	}
}

// StopFunc when called unsubscribes the subscriber from all events and
// releases resources tied to the subscriber.
type StopFunc func()

// SubscribeOption is used to provide options to Subscribe.
type SubscribeOption interface {
	// set sets the provided option.
	set(*subscribeOptions)
}

// subscribeOptions stores the provided options.
type subscribeOptions struct {
	throttle bool
	maxRep   int
}

// subscribeOption implements Option.
type subscribeOption func(*subscribeOptions)

// set implements SubscribeOption.set.
func (o subscribeOption) set(sOpts *subscribeOptions) {
	o(sOpts)
}

// MaxRepetitive when provided, instructs the system to drop repetitive
// events instead of delivering them.
// The argument maxRep indicates the maximum number of repetitive events to
// enqueue towards the subscriber.
func MaxRepetitive(maxRep int) SubscribeOption {
	return subscribeOption(func(sOpts *subscribeOptions) {
		sOpts.throttle = true
		sOpts.maxRep = maxRep
	})
}

// Subscribe subscribes to events according to the filter.
// An empty filter indicates that the subscriber wishes to receive events of
// all kinds. If the filter is non-empty, only events of the provided type will
// be sent to the subscriber.
// Returns a function that allows the subscriber to unsubscribe.
func (eds *DistributionSystem) Subscribe(filter []terminalapi.Event, cb Callback, opts ...SubscribeOption) StopFunc {
	eds.mu.Lock()
	defer eds.mu.Unlock()

	opt := &subscribeOptions{}
	for _, o := range opts {
		o.set(opt)
	}

	id := eds.nextID
	eds.nextID++
	sub := newSubscriber(filter, cb, opt)
	eds.subscribers[id] = sub

	return func() {
		eds.mu.Lock()
		defer eds.mu.Unlock()

		sub.stop()
		delete(eds.subscribers, id)
	}
}

// Processed returns the number of events that were fully processed, i.e.
// delivered to all the subscribers and their callbacks returned.
func (eds *DistributionSystem) Processed() int {
	eds.mu.Lock()
	defer eds.mu.Unlock()

	var res int
	for _, sub := range eds.subscribers {
		res += sub.processedEvents()
	}
	return res
}
