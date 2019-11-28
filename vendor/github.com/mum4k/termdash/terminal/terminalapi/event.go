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

package terminalapi

import (
	"errors"
	"fmt"
	"image"

	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/mouse"
)

// event.go defines events that can be received through the terminal API.

// Event represents an input event.
type Event interface {
	isEvent()
}

// Keyboard is the event used when a key is pressed.
// Implements terminalapi.Event.
type Keyboard struct {
	// Key is the pressed key.
	Key keyboard.Key
}

func (*Keyboard) isEvent() {}

// String implements fmt.Stringer.
func (k Keyboard) String() string {
	return fmt.Sprintf("Keyboard{Key: %v}", k.Key)
}

// Resize is the event used when the terminal was resized.
// Implements terminalapi.Event.
type Resize struct {
	// Size is the new size of the terminal.
	Size image.Point
}

func (*Resize) isEvent() {}

// String implements fmt.Stringer.
func (r Resize) String() string {
	return fmt.Sprintf("Resize{Size: %v}", r.Size)
}

// Mouse is the event used when the mouse is moved or a mouse button is
// pressed.
// Implements terminalapi.Event.
type Mouse struct {
	// Position of the mouse on the terminal.
	Position image.Point
	// Button identifies the pressed button if any.
	Button mouse.Button
}

func (*Mouse) isEvent() {}

// String implements fmt.Stringer.
func (m Mouse) String() string {
	return fmt.Sprintf("Mouse{Position: %v, Button: %v}", m.Position, m.Button)
}

// Error is an event indicating an error while processing input.
type Error string

// NewError returns a new Error event.
func NewError(e string) *Error {
	err := Error(e)
	return &err
}

// NewErrorf returns a new Error event, arguments are similar to fmt.Sprintf.
func NewErrorf(format string, args ...interface{}) *Error {
	err := Error(fmt.Sprintf(format, args...))
	return &err
}

func (*Error) isEvent() {}

// Error returns the error that occurred.
func (e *Error) Error() error {
	if e == nil || *e == "" {
		return nil
	}
	return errors.New(string(*e))
}

// String implements fmt.Stringer.
func (e Error) String() string {
	return string(e)
}
