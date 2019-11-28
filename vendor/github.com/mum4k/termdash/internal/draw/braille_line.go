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

// braille_line.go contains code that draws lines on a braille canvas.

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/canvas/braille"
	"github.com/mum4k/termdash/internal/numbers"
)

// braillePixelChange represents an action on a pixel on the braille canvas.
type braillePixelChange int

// String implements fmt.Stringer()
func (bpc braillePixelChange) String() string {
	if n, ok := braillePixelChangeNames[bpc]; ok {
		return n
	}
	return "braillePixelChangeUnknown"
}

// braillePixelChangeNames maps braillePixelChange values to human readable names.
var braillePixelChangeNames = map[braillePixelChange]string{
	braillePixelChangeSet:   "braillePixelChangeSet",
	braillePixelChangeClear: "braillePixelChangeClear",
}

const (
	braillePixelChangeUnknown braillePixelChange = iota

	braillePixelChangeSet
	braillePixelChangeClear
)

// BrailleLineOption is used to provide options to BrailleLine().
type BrailleLineOption interface {
	// set sets the provided option.
	set(*brailleLineOptions)
}

// brailleLineOptions stores the provided options.
type brailleLineOptions struct {
	cellOpts    []cell.Option
	pixelChange braillePixelChange
}

// newBrailleLineOptions returns a new brailleLineOptions instance.
func newBrailleLineOptions() *brailleLineOptions {
	return &brailleLineOptions{
		pixelChange: braillePixelChangeSet,
	}
}

// brailleLineOption implements BrailleLineOption.
type brailleLineOption func(*brailleLineOptions)

// set implements BrailleLineOption.set.
func (o brailleLineOption) set(opts *brailleLineOptions) {
	o(opts)
}

// BrailleLineCellOpts sets options on the cells that contain the line.
// Cell options on a braille canvas can only be set on the entire cell, not per
// pixel.
func BrailleLineCellOpts(cOpts ...cell.Option) BrailleLineOption {
	return brailleLineOption(func(opts *brailleLineOptions) {
		opts.cellOpts = cOpts
	})
}

// BrailleLineClearPixels changes the behavior of BrailleLine, so that it
// clears the pixels belonging to the line instead of setting them.
// Useful in order to "erase" a line from the canvas as opposed to drawing one.
func BrailleLineClearPixels() BrailleLineOption {
	return brailleLineOption(func(opts *brailleLineOptions) {
		opts.pixelChange = braillePixelChangeClear
	})
}

// BrailleLine draws an approximated line segment on the braille canvas between
// the two provided points.
// Both start and end must be valid points within the canvas. Start and end can
// be the same point in which case only one pixel will be set on the braille
// canvas.
// The start or end coordinates must not be negative.
func BrailleLine(bc *braille.Canvas, start, end image.Point, opts ...BrailleLineOption) error {
	if start.X < 0 || start.Y < 0 {
		return fmt.Errorf("the start coordinates cannot be negative, got: %v", start)
	}
	if end.X < 0 || end.Y < 0 {
		return fmt.Errorf("the end coordinates cannot be negative, got: %v", end)
	}

	opt := newBrailleLineOptions()
	for _, o := range opts {
		o.set(opt)
	}

	points := brailleLinePoints(start, end)
	for _, p := range points {
		switch opt.pixelChange {
		case braillePixelChangeSet:
			if err := bc.SetPixel(p, opt.cellOpts...); err != nil {
				return fmt.Errorf("bc.SetPixel(%v) => %v", p, err)
			}
		case braillePixelChangeClear:
			if err := bc.ClearPixel(p, opt.cellOpts...); err != nil {
				return fmt.Errorf("bc.ClearPixel(%v) => %v", p, err)
			}
		}
	}
	return nil
}

// brailleLinePoints returns the points to set when drawing the line.
func brailleLinePoints(start, end image.Point) []image.Point {
	// Implements Bresenham's line algorithm.
	// https://en.wikipedia.org/wiki/Bresenham%27s_line_algorithm

	vertProj := numbers.Abs(end.Y - start.Y)
	horizProj := numbers.Abs(end.X - start.X)
	if vertProj < horizProj {
		if start.X > end.X {
			return lineLow(end.X, end.Y, start.X, start.Y)
		}
		return lineLow(start.X, start.Y, end.X, end.Y)
	}
	if start.Y > end.Y {
		return lineHigh(end.X, end.Y, start.X, start.Y)
	}
	return lineHigh(start.X, start.Y, end.X, end.Y)
}

// lineLow returns points that create a line whose horizontal projection
// (end.X - start.X) is longer than its vertical projection
// (end.Y - start.Y).
func lineLow(x0, y0, x1, y1 int) []image.Point {
	deltaX := x1 - x0
	deltaY := y1 - y0

	stepY := 1
	if deltaY < 0 {
		stepY = -1
		deltaY = -deltaY
	}

	var res []image.Point
	diff := 2*deltaY - deltaX
	y := y0
	for x := x0; x <= x1; x++ {
		res = append(res, image.Point{x, y})
		if diff > 0 {
			y += stepY
			diff -= 2 * deltaX
		}
		diff += 2 * deltaY
	}
	return res
}

// lineHigh returns points that createa line whose vertical projection
// (end.Y - start.Y) is longer than its horizontal projection
// (end.X - start.X).
func lineHigh(x0, y0, x1, y1 int) []image.Point {
	deltaX := x1 - x0
	deltaY := y1 - y0

	stepX := 1
	if deltaX < 0 {
		stepX = -1
		deltaX = -deltaX
	}

	var res []image.Point
	diff := 2*deltaX - deltaY
	x := x0
	for y := y0; y <= y1; y++ {
		res = append(res, image.Point{x, y})

		if diff > 0 {
			x += stepX
			diff -= 2 * deltaY
		}
		diff += 2 * deltaX
	}
	return res
}
