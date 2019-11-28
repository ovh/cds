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

package draw

// text.go contains code that prints UTF-8 encoded strings on the canvas.

import (
	"fmt"
	"image"
	"strings"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/canvas"
	"github.com/mum4k/termdash/internal/runewidth"
)

// OverrunMode represents
type OverrunMode int

// String implements fmt.Stringer()
func (om OverrunMode) String() string {
	if n, ok := overrunModeNames[om]; ok {
		return n
	}
	return "OverrunModeUnknown"
}

// overrunModeNames maps OverrunMode values to human readable names.
var overrunModeNames = map[OverrunMode]string{
	OverrunModeStrict:   "OverrunModeStrict",
	OverrunModeTrim:     "OverrunModeTrim",
	OverrunModeThreeDot: "OverrunModeThreeDot",
}

const (
	// OverrunModeStrict verifies that the drawn value fits the canvas and
	// returns an error if it doesn't.
	OverrunModeStrict OverrunMode = iota

	// OverrunModeTrim trims the part of the text that doesn't fit.
	OverrunModeTrim

	// OverrunModeThreeDot trims the text and places the horizontal ellipsis
	// '…' character at the end.
	OverrunModeThreeDot
)

// TextOption is used to provide options to Text().
type TextOption interface {
	// set sets the provided option.
	set(*textOptions)
}

// textOptions stores the provided options.
type textOptions struct {
	cellOpts    []cell.Option
	maxX        int
	overrunMode OverrunMode
}

// textOption implements TextOption.
type textOption func(*textOptions)

// set implements TextOption.set.
func (to textOption) set(tOpts *textOptions) {
	to(tOpts)
}

// TextCellOpts sets options on the cells that contain the text.
func TextCellOpts(opts ...cell.Option) TextOption {
	return textOption(func(tOpts *textOptions) {
		tOpts.cellOpts = opts
	})
}

// TextMaxX sets a limit on the X coordinate (column) of the drawn text.
// The X coordinate of all cells used by the text must be within
// start.X <= X < TextMaxX.
// If not provided, the width of the canvas is used as TextMaxX.
func TextMaxX(x int) TextOption {
	return textOption(func(tOpts *textOptions) {
		tOpts.maxX = x
	})
}

// TextOverrunMode indicates what to do with text that overruns the TextMaxX()
// or the width of the canvas if TextMaxX() isn't specified.
// Defaults to OverrunModeStrict.
func TextOverrunMode(om OverrunMode) TextOption {
	return textOption(func(tOpts *textOptions) {
		tOpts.overrunMode = om
	})
}

// TrimText trims the provided text so that it fits the specified amount of cells.
func TrimText(text string, maxCells int, om OverrunMode) (string, error) {
	if maxCells < 1 {
		return "", fmt.Errorf("maxCells(%d) cannot be less than one", maxCells)
	}

	textCells := runewidth.StringWidth(text)
	if textCells <= maxCells {
		// Nothing to do if the text fits.
		return text, nil
	}

	switch om {
	case OverrunModeStrict:
		return "", fmt.Errorf("the requested text %q takes %d cells to draw, space is available for only %d cells and overrun mode is %v", text, textCells, maxCells, om)
	case OverrunModeTrim, OverrunModeThreeDot:
	default:
		return "", fmt.Errorf("unsupported overrun mode %d", om)
	}

	var b strings.Builder
	cur := 0
	for _, r := range text {
		rw := runewidth.RuneWidth(r)
		if cur+rw >= maxCells {
			switch {
			case om == OverrunModeTrim:
				// Only write the rune if it still fits, i.e. don't cut
				// full-width runes in half.
				if cur+rw == maxCells {
					b.WriteRune(r)
				}
			case om == OverrunModeThreeDot:
				b.WriteRune('…')
			}
			break
		}

		b.WriteRune(r)
		cur += rw
	}
	return b.String(), nil
}

// Text prints the provided text on the canvas starting at the provided point.
func Text(c *canvas.Canvas, text string, start image.Point, opts ...TextOption) error {
	ar := c.Area()
	if !start.In(ar) {
		return fmt.Errorf("the requested start point %v falls outside of the provided canvas %v", start, ar)
	}

	opt := &textOptions{}
	for _, o := range opts {
		o.set(opt)
	}

	if opt.maxX < 0 || opt.maxX > ar.Max.X {
		return fmt.Errorf("invalid TextMaxX(%v), must be a positive number that is <= canvas.width %v", opt.maxX, ar.Dx())
	}

	var wantMaxX int
	if opt.maxX == 0 {
		wantMaxX = ar.Max.X
	} else {
		wantMaxX = opt.maxX
	}

	maxCells := wantMaxX - start.X
	trimmed, err := TrimText(text, maxCells, opt.overrunMode)
	if err != nil {
		return err
	}

	cur := start
	for _, r := range trimmed {
		cells, err := c.SetCell(cur, r, opt.cellOpts...)
		if err != nil {
			return err
		}
		cur = image.Point{cur.X + cells, cur.Y}
	}
	return nil
}

// ResizeNeeded draws an unicode character indicating that the canvas size is
// too small to draw meaningful content.
func ResizeNeeded(cvs *canvas.Canvas) error {
	return Text(cvs, "⇄", image.Point{0, 0})
}
