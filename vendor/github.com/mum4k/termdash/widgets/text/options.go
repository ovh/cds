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

package text

import (
	"fmt"

	"github.com/mum4k/termdash/internal/wrap"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/mouse"
)

// options.go contains configurable options for Text.

// Option is used to provide options to New().
type Option interface {
	// set sets the provided option.
	set(*options)
}

// options stores the provided options.
type options struct {
	wrapMode         wrap.Mode
	rollContent      bool
	disableScrolling bool
	mouseUpButton    mouse.Button
	mouseDownButton  mouse.Button
	keyUp            keyboard.Key
	keyDown          keyboard.Key
	keyPgUp          keyboard.Key
	keyPgDown        keyboard.Key
}

// newOptions returns a new options instance.
func newOptions(opts ...Option) *options {
	opt := &options{
		mouseUpButton:   DefaultScrollMouseButtonUp,
		mouseDownButton: DefaultScrollMouseButtonDown,
		keyUp:           DefaultScrollKeyUp,
		keyDown:         DefaultScrollKeyDown,
		keyPgUp:         DefaultScrollKeyPageUp,
		keyPgDown:       DefaultScrollKeyPageDown,
	}
	for _, o := range opts {
		o.set(opt)
	}
	return opt
}

// validate validates the provided options.
func (o *options) validate() error {
	keys := map[keyboard.Key]bool{
		o.keyUp:     true,
		o.keyDown:   true,
		o.keyPgUp:   true,
		o.keyPgDown: true,
	}
	if len(keys) != 4 {
		return fmt.Errorf("invalid ScrollKeys(up:%v, down:%v, pageUp:%v, pageDown:%v), the keys must be unique", o.keyUp, o.keyDown, o.keyPgUp, o.keyPgDown)
	}
	if o.mouseUpButton == o.mouseDownButton {
		return fmt.Errorf("invalid ScrollMouseButtons(up:%v, down:%v), the buttons must be unique", o.mouseUpButton, o.mouseDownButton)
	}
	return nil
}

// option implements Option.
type option func(*options)

// set implements Option.set.
func (o option) set(opts *options) {
	o(opts)
}

// WrapAtWords configures the text widget so that it automatically wraps lines
// that are longer than the width of the widget at word boundaries. If not
// provided, long lines are trimmed instead.
func WrapAtWords() Option {
	return option(func(opts *options) {
		opts.wrapMode = wrap.AtWords
	})
}

// WrapAtRunes configures the text widget so that it automatically wraps lines
// that are longer than the width of the widget at rune boundaries. If not
// provided, long lines are trimmed instead.
func WrapAtRunes() Option {
	return option(func(opts *options) {
		opts.wrapMode = wrap.AtRunes
	})
}

// RollContent configures the text widget so that it rolls the text content up
// if more text than the size of the container is added. If not provided, the
// content is trimmed instead.
func RollContent() Option {
	return option(func(opts *options) {
		opts.rollContent = true
	})
}

// DisableScrolling disables the scrolling of the content using keyboard and
// mouse.
func DisableScrolling() Option {
	return option(func(opts *options) {
		opts.disableScrolling = true
	})
}

// The default mouse buttons for content scrolling.
const (
	DefaultScrollMouseButtonUp   = mouse.ButtonWheelUp
	DefaultScrollMouseButtonDown = mouse.ButtonWheelDown
)

// ScrollMouseButtons configures the mouse buttons that scroll the content.
// The provided buttons must be unique, e.g. the same button cannot be both up
// and down.
func ScrollMouseButtons(up, down mouse.Button) Option {
	return option(func(opts *options) {
		opts.mouseUpButton = up
		opts.mouseDownButton = down
	})
}

// The default keys for content scrolling.
const (
	DefaultScrollKeyUp       = keyboard.KeyArrowUp
	DefaultScrollKeyDown     = keyboard.KeyArrowDown
	DefaultScrollKeyPageUp   = keyboard.KeyPgUp
	DefaultScrollKeyPageDown = keyboard.KeyPgDn
)

// ScrollKeys configures the keyboard keys that scroll the content.
// The provided keys must be unique, e.g. the same key cannot be both up and
// down.
func ScrollKeys(up, down, pageUp, pageDown keyboard.Key) Option {
	return option(func(opts *options) {
		opts.keyUp = up
		opts.keyDown = down
		opts.keyPgUp = pageUp
		opts.keyPgDown = pageDown
	})
}
