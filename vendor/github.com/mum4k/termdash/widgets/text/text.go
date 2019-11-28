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

// Package text contains a widget that displays textual data.
package text

import (
	"fmt"
	"image"
	"sync"

	"github.com/mum4k/termdash/internal/canvas"
	"github.com/mum4k/termdash/internal/canvas/buffer"
	"github.com/mum4k/termdash/internal/wrap"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgetapi"
)

// Text displays a block of text.
//
// Each line of the text is either trimmed or wrapped according to the provided
// options. The entire text content is either trimmed or rolled up through the
// canvas according to the provided options.
//
// By default the widget supports scrolling of content with either the keyboard
// or mouse. See the options for the default keys and mouse buttons.
//
// Implements widgetapi.Widget. This object is thread-safe.
type Text struct {
	// content is the text content that will be displayed in the widget as
	// provided by the caller (i.e. not wrapped or pre-processed).
	content []*buffer.Cell
	// wrapped is the content wrapped to the current width of the canvas.
	wrapped [][]*buffer.Cell

	// scroll tracks scrolling the position.
	scroll *scrollTracker

	// lastWidth stores the width of the last canvas the widget drew on.
	// Used to determine if the previous line wrapping was invalidated.
	lastWidth int
	// contentChanged indicates if the text content of the widget changed since
	// the last drawing. Used to determine if the previous line wrapping was
	// invalidated.
	contentChanged bool

	// mu protects the Text widget.
	mu sync.Mutex

	// opts are the provided options.
	opts *options
}

// New returns a new text widget.
func New(opts ...Option) (*Text, error) {
	opt := newOptions(opts...)
	if err := opt.validate(); err != nil {
		return nil, err
	}
	return &Text{
		scroll: newScrollTracker(opt),
		opts:   opt,
	}, nil
}

// Reset resets the widget back to empty content.
func (t *Text) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reset()
}

// reset implements Reset, caller must hold t.mu.
func (t *Text) reset() {
	t.content = nil
	t.wrapped = nil
	t.scroll = newScrollTracker(t.opts)
	t.lastWidth = 0
	t.contentChanged = true
}

// Write writes text for the widget to display. Multiple calls append
// additional text. The text contain cannot control characters
// (unicode.IsControl) or space character (unicode.IsSpace) other than:
//   ' ', '\n'
// Any newline ('\n') characters are interpreted as newlines when displaying
// the text.
func (t *Text) Write(text string, wOpts ...WriteOption) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := wrap.ValidText(text); err != nil {
		return err
	}

	opts := newWriteOptions(wOpts...)
	if opts.replace {
		t.reset()
	}
	for _, r := range text {
		t.content = append(t.content, buffer.NewCell(r, opts.cellOpts))
	}
	t.contentChanged = true
	return nil
}

// minLinesForMarkers are the minimum amount of lines required on the canvas in
// order to draw the scroll markers ('⇧' and '⇩').
const minLinesForMarkers = 3

// drawScrollUp draws the scroll up marker on the first line if there is more
// text "above" the canvas due to the scrolling position. Returns true if the
// marker was drawn.
func (t *Text) drawScrollUp(cvs *canvas.Canvas, cur image.Point, fromLine int) (bool, error) {
	height := cvs.Area().Dy()
	if cur.Y == 0 && height >= minLinesForMarkers && fromLine > 0 {
		cells, err := cvs.SetCell(cur, '⇧')
		if err != nil {
			return false, err
		}
		if cells != 1 {
			panic(fmt.Errorf("invalid scroll up marker, it occupies %d cells, the implementation only supports scroll markers that occupy exactly one cell", cells))
		}
		return true, nil
	}
	return false, nil
}

// drawScrollDown draws the scroll down marker on the last line if there is
// more text "below" the canvas due to the scrolling position. Returns true if
// the marker was drawn.
func (t *Text) drawScrollDown(cvs *canvas.Canvas, cur image.Point, fromLine int) (bool, error) {
	height := cvs.Area().Dy()
	lines := len(t.wrapped)
	if cur.Y == height-1 && height >= minLinesForMarkers && height < lines-fromLine {
		cells, err := cvs.SetCell(cur, '⇩')
		if err != nil {
			return false, err
		}
		if cells != 1 {
			panic(fmt.Errorf("invalid scroll down marker, it occupies %d cells, the implementation only supports scroll markers that occupy exactly one cell", cells))
		}
		return true, nil
	}
	return false, nil
}

// draw draws the text context on the canvas starting at the specified line.
func (t *Text) draw(cvs *canvas.Canvas) error {
	var cur image.Point // Tracks the current drawing position on the canvas.
	height := cvs.Area().Dy()
	fromLine := t.scroll.firstLine(len(t.wrapped), height)

	for _, line := range t.wrapped[fromLine:] {
		// Scroll up marker.
		scrlUp, err := t.drawScrollUp(cvs, cur, fromLine)
		if err != nil {
			return err
		}
		if scrlUp {
			cur = image.Point{0, cur.Y + 1} // Move to the next line.
			// Skip one line of text, the marker replaced it.
			continue
		}

		// Scroll down marker.
		scrlDown, err := t.drawScrollDown(cvs, cur, fromLine)
		if err != nil {
			return err
		}
		if scrlDown || cur.Y >= height {
			break // Skip all lines falling after (under) the canvas.
		}

		for _, cell := range line {
			tr, err := lineTrim(cvs, cur, cell.Rune, t.opts)
			if err != nil {
				return err
			}
			cur = tr.curPoint
			if tr.trimmed {
				break // Skip over any characters trimmed on the current line.
			}

			cells, err := cvs.SetCell(cur, cell.Rune, cell.Opts)
			if err != nil {
				return err
			}
			cur = image.Point{cur.X + cells, cur.Y} // Move within the same line.
		}
		cur = image.Point{0, cur.Y + 1} // Move to the next line.
	}
	return nil
}

// Draw draws the text onto the canvas.
// Implements widgetapi.Widget.Draw.
func (t *Text) Draw(cvs *canvas.Canvas, meta *widgetapi.Meta) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	width := cvs.Area().Dx()
	if len(t.content) > 0 && (t.contentChanged || t.lastWidth != width) {
		// The previous text preprocessing (line wrapping) is invalidated when
		// new text is added or the width of the canvas changed.
		wr, err := wrap.Cells(t.content, width, t.opts.wrapMode)
		if err != nil {
			return err
		}
		t.wrapped = wr
	}
	t.lastWidth = width

	if len(t.wrapped) == 0 {
		return nil // Nothing to draw if there's no text.
	}

	if err := t.draw(cvs); err != nil {
		return err
	}
	t.contentChanged = false
	return nil
}

// Keyboard implements widgetapi.Widget.Keyboard.
func (t *Text) Keyboard(k *terminalapi.Keyboard) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch {
	case k.Key == t.opts.keyUp:
		t.scroll.upOneLine()
	case k.Key == t.opts.keyDown:
		t.scroll.downOneLine()
	case k.Key == t.opts.keyPgUp:
		t.scroll.upOnePage()
	case k.Key == t.opts.keyPgDown:
		t.scroll.downOnePage()
	}
	return nil
}

// Mouse implements widgetapi.Widget.Mouse.
func (t *Text) Mouse(m *terminalapi.Mouse) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch b := m.Button; {
	case b == t.opts.mouseUpButton:
		t.scroll.upOneLine()
	case b == t.opts.mouseDownButton:
		t.scroll.downOneLine()
	}
	return nil
}

// Options of the widget
func (t *Text) Options() widgetapi.Options {
	var ks widgetapi.KeyScope
	var ms widgetapi.MouseScope
	if t.opts.disableScrolling {
		ks = widgetapi.KeyScopeNone
		ms = widgetapi.MouseScopeNone
	} else {
		ks = widgetapi.KeyScopeFocused
		ms = widgetapi.MouseScopeWidget
	}

	return widgetapi.Options{
		// At least one line with at least one full-width rune.
		MinimumSize:  image.Point{1, 1},
		WantMouse:    ms,
		WantKeyboard: ks,
	}
}
