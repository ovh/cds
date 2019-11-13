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

// Package button implements a state machine that tracks mouse button clicks.
package button

import (
	"image"

	"github.com/mum4k/termdash/mouse"
	"github.com/mum4k/termdash/terminal/terminalapi"
)

// State represents the state of the mouse button.
type State int

// String implements fmt.Stringer()
func (s State) String() string {
	if n, ok := stateNames[s]; ok {
		return n
	}
	return "StateUnknown"
}

// stateNames maps State values to human readable names.
var stateNames = map[State]string{
	Up:   "StateUp",
	Down: "StateDown",
}

const (
	// Up is the default idle state of the mouse button.
	Up State = iota

	// Down is a state where the mouse button is pressed down and held.
	Down
)

// FSM implements a finite-state machine that tracks mouse clicks within an
// area.
//
// Simplifies tracking of mouse button clicks, i.e. when the caller wants to
// perform an action only if both the button press and release happen within
// the specified area.
//
// This object is not thread-safe.
type FSM struct {
	// button is the mouse button whose state this FSM tracks.
	button mouse.Button

	// area is the area provided to NewFSM.
	area image.Rectangle

	// state is the current state of the FSM.
	state stateFn
}

// NewFSM creates a new FSM instance that tracks the state of the specified
// mouse button through button events that fall within the provided area.
func NewFSM(button mouse.Button, area image.Rectangle) *FSM {
	return &FSM{
		button: button,
		area:   area,
		state:  wantPress,
	}
}

// Event is used to forward mouse events to the state machine.
// Only events related to the button specified on a call to NewFSM are
// processed.
//
// Returns a bool indicating if an action guarded by the button should be
// performed and the state of the button after the provided event.
// The bool is true if the button click should take an effect, i.e. if the
// FSM saw both the button click and its release.
func (fsm *FSM) Event(m *terminalapi.Mouse) (bool, State) {
	clicked, bs, next := fsm.state(fsm, m)
	fsm.state = next
	return clicked, bs
}

// UpdateArea informs FSM of an area change.
// This method is idempotent.
func (fsm *FSM) UpdateArea(area image.Rectangle) {
	fsm.area = area
}

// stateFn is a single state in the state machine.
// Returns bool indicating if a click happened, the state of the button and the
// next state of the FSM.
type stateFn func(fsm *FSM, m *terminalapi.Mouse) (bool, State, stateFn)

// wantPress is the initial state, expecting a button press inside the area.
func wantPress(fsm *FSM, m *terminalapi.Mouse) (bool, State, stateFn) {
	if m.Button != fsm.button || !m.Position.In(fsm.area) {
		return false, Up, wantPress
	}
	return false, Down, wantRelease
}

// wantRelease waits for a mouse button release in the same area as
// the press.
func wantRelease(fsm *FSM, m *terminalapi.Mouse) (bool, State, stateFn) {
	switch m.Button {
	case fsm.button:
		if m.Position.In(fsm.area) {
			// Remain in the same state, since termbox reports move of mouse with
			// button held down as a series of clicks, one per position.
			return false, Down, wantRelease
		}
		return false, Up, wantPress

	case mouse.ButtonRelease:
		if m.Position.In(fsm.area) {
			// Seen both press and release, report a click.
			return true, Up, wantPress
		}
		// Release the button even if the release event happened outside of the area.
		return false, Up, wantPress

	default:
		return false, Up, wantPress
	}
}
