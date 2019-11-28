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

// Package widgetapi defines the API of a widget on the dashboard.
package widgetapi

import (
	"image"

	"github.com/mum4k/termdash/internal/canvas"
	"github.com/mum4k/termdash/terminal/terminalapi"
)

// KeyScope indicates the scope at which the widget wants to receive keyboard
// events.
type KeyScope int

// String implements fmt.Stringer()
func (ks KeyScope) String() string {
	if n, ok := keyScopeNames[ks]; ok {
		return n
	}
	return "KeyScopeUnknown"
}

// keyScopeNames maps KeyScope values to human readable names.
var keyScopeNames = map[KeyScope]string{
	KeyScopeNone:    "KeyScopeNone",
	KeyScopeFocused: "KeyScopeFocused",
	KeyScopeGlobal:  "KeyScopeGlobal",
}

const (
	// KeyScopeNone is used when the widget doesn't want to receive any
	// keyboard events.
	KeyScopeNone KeyScope = iota

	// KeyScopeFocused is used when the widget wants to only receive keyboard
	// events when its container is focused.
	KeyScopeFocused

	// KeyScopeGlobal is used when the widget wants to receive all keyboard
	// events regardless of which container is focused.
	KeyScopeGlobal
)

// MouseScope indicates the scope at which the widget wants to receive mouse
// events.
type MouseScope int

// String implements fmt.Stringer()
func (ms MouseScope) String() string {
	if n, ok := mouseScopeNames[ms]; ok {
		return n
	}
	return "MouseScopeUnknown"
}

// mouseScopeNames maps MouseScope values to human readable names.
var mouseScopeNames = map[MouseScope]string{
	MouseScopeNone:      "MouseScopeNone",
	MouseScopeWidget:    "MouseScopeWidget",
	MouseScopeContainer: "MouseScopeContainer",
	MouseScopeGlobal:    "MouseScopeGlobal",
}

const (
	// MouseScopeNone is used when the widget doesn't want to receive any mouse
	// events.
	MouseScopeNone MouseScope = iota

	// MouseScopeWidget is used when the widget only wants mouse events that
	// fall onto its canvas.
	// The position of these widgets is always relative to widget's canvas.
	MouseScopeWidget

	// MouseScopeContainer is used when the widget only wants mouse events that
	// fall onto its container. The area size of a container is always larger
	// or equal to the one of the widget's canvas. So a widget selecting
	// MouseScopeContainer will either receive the same or larger amount of
	// events as compared to MouseScopeWidget.
	// The position of mouse events that fall outside of widget's canvas is
	// reset to image.Point{-1, -1}.
	// The widgets are allowed to process the button event.
	MouseScopeContainer

	// MouseScopeGlobal is used when the widget wants to receive all mouse
	// events regardless on where on the terminal they land.
	// The position of mouse events that fall outside of widget's canvas is
	// reset to image.Point{-1, -1} and must not be used by the widgets.
	// The widgets are allowed to process the button event.
	MouseScopeGlobal
)

// Options contains registration options for a widget.
// This is how the widget indicates its needs to the infrastructure.
type Options struct {
	// Ratio allows a widget to request a canvas whose size will always have
	// the specified ratio of width:height (Ratio.X:Ratio.Y).
	// The zero value i.e. image.Point{0, 0} indicates that the widget accepts
	// canvas of any ratio.
	Ratio image.Point

	// MinimumSize allows a widget to specify the smallest allowed canvas size.
	// If the terminal size and/or splits cause the assigned canvas to be
	// smaller than this, the widget will be skipped. I.e. The Draw() method
	// won't be called until a resize above the specified minimum.
	MinimumSize image.Point

	// MaximumSize allows a widget to specify the largest allowed canvas size.
	// If the terminal size and/or splits cause the assigned canvas to be larger
	// than this, the widget will only receive a canvas of this size within its
	// container. Setting any of the two coordinates to zero indicates
	// unlimited.
	MaximumSize image.Point

	// WantKeyboard allows a widget to request keyboard events and specify
	// their desired scope. If set to KeyScopeNone, no keyboard events are
	// forwarded to the widget.
	WantKeyboard KeyScope

	// WantMouse allows a widget to request mouse events and specify their
	// desired scope. If set to MouseScopeNone, no mouse events are forwarded
	// to the widget.
	// Note that the widget is only able to see the position of the mouse event
	// if it falls onto its canvas. See the documentation next to individual
	// MouseScope values for details.
	WantMouse MouseScope
}

// Meta provide additional metadata to widgets.
type Meta struct {
	// Focused asserts whether the widget's container is focused.
	Focused bool
}

// Widget is a single widget on the dashboard.
// Implementations must be thread safe.
type Widget interface {
	// When the infrastructure calls Draw(), the widget must block on the call
	// until it finishes drawing onto the provided canvas. When given the
	// canvas, the widget must first determine its size by calling
	// Canvas.Size(), then limit all its drawing to this area.
	//
	// The widget must not assume that the size of the canvas or its content
	// remains the same between calls.
	//
	// The argument meta is guaranteed to be valid (i.e. non-nil).
	Draw(cvs *canvas.Canvas, meta *Meta) error

	// Keyboard is called when the widget is focused on the dashboard and a key
	// shortcut the widget registered for was pressed. Only called if the widget
	// registered for keyboard events.
	Keyboard(k *terminalapi.Keyboard) error

	// Mouse is called when the widget is focused on the dashboard and a mouse
	// event happens on its canvas. Only called if the widget registered for mouse
	// events.
	Mouse(m *terminalapi.Mouse) error

	// Options returns registration options for the widget.
	// This is how the widget indicates to the infrastructure whether it is
	// interested in keyboard or mouse shortcuts, what is its minimum canvas
	// size, etc.
	//
	// Most widgets will return statically compiled options (minimum and
	// maximum size, etc.). If the returned options depend on the runtime state
	// of the widget (e.g. the user data provided to the widget), the widget
	// must protect against a case where the infrastructure calls the Draw
	// method with a canvas that doesn't meet the requested options. This is
	// because the data in the widget might change between calls to Options and
	// Draw.
	Options() Options
}
