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

/*
Package termdash implements a terminal based dashboard.

While running, the terminal dashboard performs the following:
  - Periodic redrawing of the canvas and all the widgets.
  - Event based redrawing of the widgets (i.e. on Keyboard or Mouse events).
  - Forwards input events to widgets and optional subscribers.
  - Handles terminal resize events.
*/
package termdash

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/internal/event"
	"github.com/mum4k/termdash/terminal/terminalapi"
)

// DefaultRedrawInterval is the default for the RedrawInterval option.
const DefaultRedrawInterval = 250 * time.Millisecond

// Option is used to provide options.
type Option interface {
	// set sets the provided option.
	set(td *termdash)
}

// option implements Option.
type option func(td *termdash)

// set implements Option.set.
func (o option) set(td *termdash) {
	o(td)
}

// RedrawInterval sets how often termdash redraws the container and all the widgets.
// Defaults to DefaultRedrawInterval. Use the controller to disable the
// periodic redraw.
func RedrawInterval(t time.Duration) Option {
	return option(func(td *termdash) {
		td.redrawInterval = t
	})
}

// ErrorHandler is used to provide a function that will be called with all
// errors that occur while the dashboard is running. If not provided, any
// errors panic the application.
// The provided function must be thread-safe.
func ErrorHandler(f func(error)) Option {
	return option(func(td *termdash) {
		td.errorHandler = f
	})
}

// KeyboardSubscriber registers a subscriber for Keyboard events. Each
// keyboard event is forwarded to the container and the registered subscriber.
// The provided function must be thread-safe.
func KeyboardSubscriber(f func(*terminalapi.Keyboard)) Option {
	return option(func(td *termdash) {
		td.keyboardSubscriber = f
	})
}

// MouseSubscriber registers a subscriber for Mouse events. Each mouse event
// is forwarded to the container and the registered subscriber.
// The provided function must be thread-safe.
func MouseSubscriber(f func(*terminalapi.Mouse)) Option {
	return option(func(td *termdash) {
		td.mouseSubscriber = f
	})
}

// withEDS indicates that termdash should run with the provided event
// distribution system instead of creating one.
// Useful for tests.
func withEDS(eds *event.DistributionSystem) Option {
	return option(func(td *termdash) {
		td.eds = eds
	})
}

// Run runs the terminal dashboard with the provided container on the terminal.
// Redraws the terminal periodically. If you prefer a manual redraw, use the
// Controller instead.
// Blocks until the context expires.
func Run(ctx context.Context, t terminalapi.Terminal, c *container.Container, opts ...Option) error {
	td := newTermdash(t, c, opts...)

	err := td.start(ctx)
	// Only return the status (error or nil) after the termdash event
	// processing goroutine actually exits.
	td.stop()
	return err
}

// Controller controls a termdash instance.
// The controller instance is only valid until Close() is called.
// The controller is not thread-safe.
type Controller struct {
	td     *termdash
	cancel context.CancelFunc
}

// NewController initializes termdash and returns an instance of the controller.
// Periodic redrawing is disabled when using the controller, the RedrawInterval
// option is ignored.
// Close the controller when it isn't needed anymore.
func NewController(t terminalapi.Terminal, c *container.Container, opts ...Option) (*Controller, error) {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := &Controller{
		td:     newTermdash(t, c, opts...),
		cancel: cancel,
	}

	// stops when Close() is called.
	go ctrl.td.processEvents(ctx)
	if err := ctrl.td.periodicRedraw(); err != nil {
		return nil, err
	}
	return ctrl, nil
}

// Redraw triggers redraw of the terminal.
func (c *Controller) Redraw() error {
	if c.td == nil {
		return errors.New("the termdash instance is no longer running, this controller is now invalid")
	}

	c.td.mu.Lock()
	defer c.td.mu.Unlock()
	return c.td.redraw()
}

// Close closes the Controller and its termdash instance.
func (c *Controller) Close() {
	c.cancel()
	c.td.stop()
	c.td = nil
}

// termdash is a terminal based dashboard.
// This object is thread-safe.
type termdash struct {
	// term is the terminal the dashboard runs on.
	term terminalapi.Terminal

	// container maintains terminal splits and places widgets.
	container *container.Container

	// eds distributes input events to subscribers.
	eds *event.DistributionSystem

	// closeCh gets closed when Stop() is called, which tells the event
	// collecting goroutine to exit.
	closeCh chan struct{}
	// exitCh gets closed when the event collecting goroutine actually exits.
	exitCh chan struct{}

	// clearNeeded indicates if the terminal needs to be cleared next time
	// we're drawing it. Terminal needs to be cleared if its sized changed.
	clearNeeded bool

	// mu protects termdash.
	mu sync.Mutex

	// Options.
	redrawInterval     time.Duration
	errorHandler       func(error)
	mouseSubscriber    func(*terminalapi.Mouse)
	keyboardSubscriber func(*terminalapi.Keyboard)
}

// newTermdash creates a new termdash.
func newTermdash(t terminalapi.Terminal, c *container.Container, opts ...Option) *termdash {
	td := &termdash{
		term:           t,
		container:      c,
		eds:            event.NewDistributionSystem(),
		closeCh:        make(chan struct{}),
		exitCh:         make(chan struct{}),
		redrawInterval: DefaultRedrawInterval,
	}

	for _, opt := range opts {
		opt.set(td)
	}
	td.subscribers()
	c.Subscribe(td.eds)
	return td
}

// subscribers subscribes event receivers that live in this package to EDS.
func (td *termdash) subscribers() {
	// Handler for all errors that occur during input event processing.
	td.eds.Subscribe([]terminalapi.Event{terminalapi.NewError("")}, func(ev terminalapi.Event) {
		td.handleError(ev.(*terminalapi.Error).Error())
	})

	// Handles terminal resize events.
	td.eds.Subscribe([]terminalapi.Event{&terminalapi.Resize{}}, func(terminalapi.Event) {
		td.setClearNeeded()
	})

	// Redraws the screen on Keyboard and Mouse events.
	// These events very likely change the content of the widgets (e.g. zooming
	// a LineChart) so a redraw is needed to make that visible.
	td.eds.Subscribe([]terminalapi.Event{
		&terminalapi.Keyboard{},
		&terminalapi.Mouse{},
	}, func(terminalapi.Event) {
		td.evRedraw()
	}, event.MaxRepetitive(0)) // No repetitive events that cause terminal redraw.

	// Keyboard and Mouse subscribers specified via options.
	if td.keyboardSubscriber != nil {
		td.eds.Subscribe([]terminalapi.Event{&terminalapi.Keyboard{}}, func(ev terminalapi.Event) {
			td.keyboardSubscriber(ev.(*terminalapi.Keyboard))
		})
	}
	if td.mouseSubscriber != nil {
		td.eds.Subscribe([]terminalapi.Event{&terminalapi.Mouse{}}, func(ev terminalapi.Event) {
			td.mouseSubscriber(ev.(*terminalapi.Mouse))
		})
	}
}

// handleError forwards the error to the error handler if one was
// provided or panics.
func (td *termdash) handleError(err error) {
	if td.errorHandler != nil {
		td.errorHandler(err)
	} else {
		panic(err)
	}
}

// setClearNeeded flags that the terminal needs to be cleared next time we're
// drawing it.
func (td *termdash) setClearNeeded() {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.clearNeeded = true
}

// redraw redraws the container and its widgets.
// The caller must hold td.mu.
func (td *termdash) redraw() error {
	if td.clearNeeded {
		if err := td.term.Clear(); err != nil {
			return fmt.Errorf("term.Clear => error: %v", err)
		}
		td.clearNeeded = false
	}

	if err := td.container.Draw(); err != nil {
		return fmt.Errorf("container.Draw => error: %v", err)
	}

	if err := td.term.Flush(); err != nil {
		return fmt.Errorf("term.Flush => error: %v", err)
	}
	return nil
}

// evRedraw redraws the container and its widgets.
func (td *termdash) evRedraw() error {
	td.mu.Lock()
	defer td.mu.Unlock()

	// Don't redraw immediately, give widgets that are performing enough time
	// to update.
	// We don't want to actually synchronize until all widgets update, we are
	// purposefully leaving slow widgets behind.
	time.Sleep(25 * time.Millisecond)
	return td.redraw()
}

// periodicRedraw is called once each RedrawInterval.
func (td *termdash) periodicRedraw() error {
	td.mu.Lock()
	defer td.mu.Unlock()
	return td.redraw()
}

// processEvents processes terminal input events.
// This is the body of the event collecting goroutine.
func (td *termdash) processEvents(ctx context.Context) {
	defer close(td.exitCh)

	for {
		ev := td.term.Event(ctx)
		if ev != nil {
			td.eds.Event(ev)
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// start starts the terminal dashboard. Blocks until the context expires or
// until stop() is called.
func (td *termdash) start(ctx context.Context) error {
	// Redraw once to initialize the container sizes.
	if err := td.periodicRedraw(); err != nil {
		close(td.exitCh)
		return err
	}

	redrawTimer := time.NewTicker(td.redrawInterval)
	defer redrawTimer.Stop()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// stops when stop() is called or the context expires.
	go td.processEvents(ctx)

	for {
		select {
		case <-redrawTimer.C:
			if err := td.periodicRedraw(); err != nil {
				return err
			}

		case <-ctx.Done():
			return nil

		case <-td.closeCh:
			return nil
		}
	}
}

// stop tells the event collecting goroutine to stop.
// Blocks until it exits.
func (td *termdash) stop() {
	close(td.closeCh)
	<-td.exitCh
}
