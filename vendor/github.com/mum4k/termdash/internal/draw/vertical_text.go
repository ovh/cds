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

package draw

// vertical_text.go contains code that prints UTF-8 encoded strings on the
// canvas in vertical columns instead of lines.

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/canvas"
)

// VerticalTextOption is used to provide options to Text().
type VerticalTextOption interface {
	// set sets the provided option.
	set(*verticalTextOptions)
}

// verticalTextOptions stores the provided options.
type verticalTextOptions struct {
	cellOpts    []cell.Option
	maxY        int
	overrunMode OverrunMode
}

// verticalTextOption implements VerticalTextOption.
type verticalTextOption func(*verticalTextOptions)

// set implements VerticalTextOption.set.
func (vto verticalTextOption) set(vtOpts *verticalTextOptions) {
	vto(vtOpts)
}

// VerticalTextCellOpts sets options on the cells that contain the text.
func VerticalTextCellOpts(opts ...cell.Option) VerticalTextOption {
	return verticalTextOption(func(vtOpts *verticalTextOptions) {
		vtOpts.cellOpts = opts
	})
}

// VerticalTextMaxY sets a limit on the Y coordinate (row) of the drawn text.
// The Y coordinate of all cells used by the vertical text must be within
// start.Y <= Y < VerticalTextMaxY.
// If not provided, the height of the canvas is used as VerticalTextMaxY.
func VerticalTextMaxY(y int) VerticalTextOption {
	return verticalTextOption(func(vtOpts *verticalTextOptions) {
		vtOpts.maxY = y
	})
}

// VerticalTextOverrunMode indicates what to do with text that overruns the
// VerticalTextMaxY() or the width of the canvas if VerticalTextMaxY() isn't
// specified.
// Defaults to OverrunModeStrict.
func VerticalTextOverrunMode(om OverrunMode) VerticalTextOption {
	return verticalTextOption(func(vtOpts *verticalTextOptions) {
		vtOpts.overrunMode = om
	})
}

// VerticalText prints the provided text on the canvas starting at the provided point.
// The text is printed in a vertical orientation, i.e:
//   H
//   e
//   l
//   l
//   o
func VerticalText(c *canvas.Canvas, text string, start image.Point, opts ...VerticalTextOption) error {
	ar := c.Area()
	if !start.In(ar) {
		return fmt.Errorf("the requested start point %v falls outside of the provided canvas %v", start, ar)
	}

	opt := &verticalTextOptions{}
	for _, o := range opts {
		o.set(opt)
	}

	if opt.maxY < 0 || opt.maxY > ar.Max.Y {
		return fmt.Errorf("invalid VerticalTextMaxY(%v), must be a positive number that is <= canvas.width %v", opt.maxY, ar.Dy())
	}

	var wantMaxY int
	if opt.maxY == 0 {
		wantMaxY = ar.Max.Y
	} else {
		wantMaxY = opt.maxY
	}

	maxCells := wantMaxY - start.Y
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
		cur = image.Point{cur.X, cur.Y + cells}
	}
	return nil
}
