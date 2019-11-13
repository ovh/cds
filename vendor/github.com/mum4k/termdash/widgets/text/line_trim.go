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
	"image"

	"github.com/mum4k/termdash/internal/canvas"
	"github.com/mum4k/termdash/internal/runewidth"
	"github.com/mum4k/termdash/internal/wrap"
)

// line_trim.go contains code that trims lines that are too long.

type trimResult struct {
	// trimmed is set to true if the current and the following runes on this
	// line are trimmed.
	trimmed bool

	// curPoint is the updated current point the drawing should continue on.
	curPoint image.Point
}

// drawTrimChar draws the horizontal ellipsis '…' character as the last
// character in the canvas on the specified line.
func drawTrimChar(cvs *canvas.Canvas, line int) error {
	lastPoint := image.Point{cvs.Area().Dx() - 1, line}
	// If the penultimate cell contains a full-width rune, we need to clear it
	// first. Otherwise the trim char would cover just half of it.
	if width := cvs.Area().Dx(); width > 1 {
		penUlt := image.Point{width - 2, line}
		prev, err := cvs.Cell(penUlt)
		if err != nil {
			return err
		}

		if runewidth.RuneWidth(prev.Rune) == 2 {
			if _, err := cvs.SetCell(penUlt, 0); err != nil {
				return err
			}
		}
	}

	cells, err := cvs.SetCell(lastPoint, '…')
	if err != nil {
		return err
	}
	if cells != 1 {
		panic(fmt.Errorf("invalid trim character, it occupies %d cells, the implementation only supports scroll markers that occupy exactly one cell", cells))
	}
	return nil
}

// lineTrim determines if the current line needs to be trimmed. The cvs is the
// canvas assigned to the widget, the curPoint is the current point the widget
// is going to place the curRune at. If line trimming is needed, this function
// replaces the last character with the horizontal ellipsis '…' character.
func lineTrim(cvs *canvas.Canvas, curPoint image.Point, curRune rune, opts *options) (*trimResult, error) {
	if opts.wrapMode == wrap.AtRunes {
		// Don't trim if the widget is configured to wrap lines.
		return &trimResult{
			trimmed:  false,
			curPoint: curPoint,
		}, nil
	}

	// Newline characters are never trimmed, they start the next line.
	if curRune == '\n' {
		return &trimResult{
			trimmed:  false,
			curPoint: curPoint,
		}, nil
	}

	width := cvs.Area().Dx()
	rw := runewidth.RuneWidth(curRune)
	switch {
	case rw == 1:
		if curPoint.X == width {
			if err := drawTrimChar(cvs, curPoint.Y); err != nil {
				return nil, err
			}
		}

	case rw == 2:
		if curPoint.X == width || curPoint.X == width-1 {
			if err := drawTrimChar(cvs, curPoint.Y); err != nil {
				return nil, err
			}
		}

	default:
		return nil, fmt.Errorf("unable to decide line trimming at position %v for rune %q which has an unsupported width %d", curPoint, curRune, rw)
	}

	trimmed := curPoint.X > width-rw
	if trimmed {
		curPoint = image.Point{curPoint.X + rw, curPoint.Y}
	}
	return &trimResult{
		trimmed:  trimmed,
		curPoint: curPoint,
	}, nil
}
