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

// Package termbox implements terminal using the nsf/termbox-go library.
package termbox

import (
	"context"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/event/eventqueue"
	"github.com/mum4k/termdash/terminal/terminalapi"
	tbx "github.com/nsf/termbox-go"
)

// Option is used to provide options.
type Option interface {
	// set sets the provided option.
	set(*Terminal)
}

// option implements Option.
type option func(*Terminal)

// set implements Option.set.
func (o option) set(t *Terminal) {
	o(t)
}

// DefaultColorMode is the default value for the ColorMode option.
const DefaultColorMode = terminalapi.ColorMode256

// ColorMode sets the terminal color mode.
// Defaults to DefaultColorMode.
func ColorMode(cm terminalapi.ColorMode) Option {
	return option(func(t *Terminal) {
		t.colorMode = cm
	})
}

// Terminal provides input and output to a real terminal. Wraps the
// nsf/termbox-go terminal implementation. This object is not thread-safe.
// Implements terminalapi.Terminal.
type Terminal struct {
	// events is a queue of input events.
	events *eventqueue.Unbound

	// done gets closed when Close() is called.
	done chan struct{}

	// Options.
	colorMode terminalapi.ColorMode
}

// newTerminal creates the terminal and applies the options.
func newTerminal(opts ...Option) *Terminal {
	t := &Terminal{
		events:    eventqueue.New(),
		done:      make(chan struct{}),
		colorMode: DefaultColorMode,
	}
	for _, opt := range opts {
		opt.set(t)
	}
	return t
}

// New returns a new termbox based Terminal.
// Call Close() when the terminal isn't required anymore.
func New(opts ...Option) (*Terminal, error) {
	if err := tbx.Init(); err != nil {
		return nil, err
	}
	tbx.SetInputMode(tbx.InputEsc | tbx.InputMouse)

	t := newTerminal(opts...)
	om, err := colorMode(t.colorMode)
	if err != nil {
		return nil, err
	}
	tbx.SetOutputMode(om)

	go t.pollEvents() // Stops when Close() is called.
	return t, nil
}

// Size implements terminalapi.Terminal.Size.
func (t *Terminal) Size() image.Point {
	w, h := tbx.Size()
	return image.Point{w, h}
}

// Clear implements terminalapi.Terminal.Clear.
func (t *Terminal) Clear(opts ...cell.Option) error {
	o := cell.NewOptions(opts...)
	return tbx.Clear(cellOptsToFg(o), cellOptsToBg(o))
}

// Flush implements terminalapi.Terminal.Flush.
func (t *Terminal) Flush() error {
	return tbx.Flush()
}

// SetCursor implements terminalapi.Terminal.SetCursor.
func (t *Terminal) SetCursor(p image.Point) {
	tbx.SetCursor(p.X, p.Y)
}

// HideCursor implements terminalapi.Terminal.HideCursor.
func (t *Terminal) HideCursor() {
	tbx.HideCursor()
}

// SetCell implements terminalapi.Terminal.SetCell.
func (t *Terminal) SetCell(p image.Point, r rune, opts ...cell.Option) error {
	o := cell.NewOptions(opts...)
	tbx.SetCell(p.X, p.Y, r, cellOptsToFg(o), cellOptsToBg(o))
	return nil
}

// pollEvents polls and enqueues the input events.
func (t *Terminal) pollEvents() {
	for {
		select {
		case <-t.done:
			return
		default:
		}

		events := toTermdashEvents(tbx.PollEvent())
		for _, ev := range events {
			t.events.Push(ev)
		}
	}
}

// Event implements terminalapi.Terminal.Event.
func (t *Terminal) Event(ctx context.Context) terminalapi.Event {
	ev := t.events.Pull(ctx)
	if ev == nil {
		return nil
	}
	return ev
}

// Close closes the terminal, should be called when the terminal isn't required
// anymore to return the screen to a sane state.
// Implements terminalapi.Terminal.Close.
func (t *Terminal) Close() {
	close(t.done)
	tbx.Close()
}
