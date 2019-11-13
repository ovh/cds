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

// Package canvas defines the canvas that the widgets draw on.
package canvas

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/area"
	"github.com/mum4k/termdash/internal/canvas/buffer"
	"github.com/mum4k/termdash/internal/runewidth"
	"github.com/mum4k/termdash/terminal/terminalapi"
)

// Canvas is where a widget draws its output for display on the terminal.
type Canvas struct {
	// area is the area the buffer was created for.
	// Contains absolute coordinates on the target terminal, while the buffer
	// contains relative zero-based coordinates for this canvas.
	area image.Rectangle

	// buffer is where the drawing happens.
	buffer buffer.Buffer
}

// New returns a new Canvas with a buffer for the provided area.
func New(ar image.Rectangle) (*Canvas, error) {
	if ar.Min.X < 0 || ar.Min.Y < 0 || ar.Max.X < 0 || ar.Max.Y < 0 {
		return nil, fmt.Errorf("area cannot start or end on the negative axis, got: %+v", ar)
	}

	b, err := buffer.New(area.Size(ar))
	if err != nil {
		return nil, err
	}
	return &Canvas{
		area:   ar,
		buffer: b,
	}, nil
}

// Size returns the size of the 2-D canvas.
func (c *Canvas) Size() image.Point {
	return c.buffer.Size()
}

// Area returns the area of the 2-D canvas.
func (c *Canvas) Area() image.Rectangle {
	s := c.buffer.Size()
	return image.Rect(0, 0, s.X, s.Y)
}

// Clear clears all the content on the canvas.
func (c *Canvas) Clear() error {
	b, err := buffer.New(c.Size())
	if err != nil {
		return err
	}
	c.buffer = b
	return nil
}

// SetCell sets the rune of the specified cell on the canvas. Returns the
// number of cells the rune occupies, wide runes can occupy multiple cells when
// printed on the terminal. See http://www.unicode.org/reports/tr11/.
// Use the options to specify which attributes to modify, if an attribute
// option isn't specified, the attribute retains its previous value.
func (c *Canvas) SetCell(p image.Point, r rune, opts ...cell.Option) (int, error) {
	return c.buffer.SetCell(p, r, opts...)
}

// Cell returns a copy of the specified cell.
func (c *Canvas) Cell(p image.Point) (*buffer.Cell, error) {
	ar, err := area.FromSize(c.Size())
	if err != nil {
		return nil, err
	}
	if !p.In(ar) {
		return nil, fmt.Errorf("point %v falls outside of the area %v occupied by the canvas", p, ar)
	}

	return c.buffer[p.X][p.Y].Copy(), nil
}

// SetCellOpts sets options on the specified cell of the canvas without
// modifying the content of the cell.
// Sets the default cell options if no options are provided.
// This method is idempotent.
func (c *Canvas) SetCellOpts(p image.Point, opts ...cell.Option) error {
	curCell, err := c.Cell(p)
	if err != nil {
		return err
	}

	if len(opts) == 0 {
		// Set the default options.
		opts = []cell.Option{
			cell.FgColor(cell.ColorDefault),
			cell.BgColor(cell.ColorDefault),
		}
	}
	if _, err := c.SetCell(p, curCell.Rune, opts...); err != nil {
		return err
	}
	return nil
}

// SetAreaCells is like SetCell, but sets the specified rune and options on all
// the cells within the provided area.
// This method is idempotent.
func (c *Canvas) SetAreaCells(cellArea image.Rectangle, r rune, opts ...cell.Option) error {
	haveArea := c.Area()
	if !cellArea.In(haveArea) {
		return fmt.Errorf("unable to set cell runes in area %v, it must fit inside the available cell area is %v", cellArea, haveArea)
	}

	rw := runewidth.RuneWidth(r)
	for row := cellArea.Min.Y; row < cellArea.Max.Y; row++ {
		for col := cellArea.Min.X; col < cellArea.Max.X; {
			p := image.Point{col, row}
			if col+rw > cellArea.Max.X {
				break
			}
			cells, err := c.SetCell(p, r, opts...)
			if err != nil {
				return err
			}
			col += cells
		}
	}
	return nil
}

// SetAreaCellOpts is like SetCellOpts, but sets the specified options on all
// the cells within the provided area.
func (c *Canvas) SetAreaCellOpts(cellArea image.Rectangle, opts ...cell.Option) error {
	haveArea := c.Area()
	if !cellArea.In(haveArea) {
		return fmt.Errorf("unable to set cell options in area %v, it must fit inside the available cell area is %v", cellArea, haveArea)
	}
	for col := cellArea.Min.X; col < cellArea.Max.X; col++ {
		for row := cellArea.Min.Y; row < cellArea.Max.Y; row++ {
			if err := c.SetCellOpts(image.Point{col, row}, opts...); err != nil {
				return err
			}
		}
	}
	return nil
}

// setCellFunc is a function that sets cell content on a terminal or a canvas.
type setCellFunc func(image.Point, rune, ...cell.Option) error

// copyTo is the internal implementation of code that copies the content of a
// canvas. If a non zero offset is provided, all the copied points are offset by
// this amount.
// The dstSetCell function is called for every point in this canvas when
// copying it to the destination.
func (c *Canvas) copyTo(offset image.Point, dstSetCell setCellFunc) error {
	for col := range c.buffer {
		for row := range c.buffer[col] {
			partial, err := c.buffer.IsPartial(image.Point{col, row})
			if err != nil {
				return err
			}
			if partial {
				// Skip over partial cells, i.e. cells that follow a cell
				// containing a full-width rune. A full-width rune takes only
				// one cell in the buffer, but two on the terminal.
				// See http://www.unicode.org/reports/tr11/.
				continue
			}
			cell := c.buffer[col][row]
			p := image.Point{col, row}.Add(offset)
			if err := dstSetCell(p, cell.Rune, cell.Opts); err != nil {
				return fmt.Errorf("setCellFunc%v => error: %v", p, err)
			}
		}
	}
	return nil
}

// Apply applies the canvas to the corresponding area of the terminal.
// Guarantees to stay within limits of the area the canvas was created with.
func (c *Canvas) Apply(t terminalapi.Terminal) error {
	termArea, err := area.FromSize(t.Size())
	if err != nil {
		return err
	}

	bufArea, err := area.FromSize(c.buffer.Size())
	if err != nil {
		return err
	}

	if !bufArea.In(termArea) {
		return fmt.Errorf("the canvas area %+v doesn't fit onto the terminal %+v", bufArea, termArea)
	}

	// The image.Point{0, 0} of this canvas isn't always exactly at
	// image.Point{0, 0} on the terminal.
	// Depends on area assigned by the container.
	offset := c.area.Min
	return c.copyTo(offset, t.SetCell)
}

// CopyTo copies the content of this canvas onto the destination canvas.
// This canvas can have an offset when compared to the destination canvas, i.e.
// the area of this canvas doesn't have to be zero-based.
func (c *Canvas) CopyTo(dst *Canvas) error {
	if !c.area.In(dst.Area()) {
		return fmt.Errorf("the canvas area %v doesn't fit or lie inside the destination canvas area %v", c.area, dst.Area())
	}

	fn := setCellFunc(func(p image.Point, r rune, opts ...cell.Option) error {
		if _, err := dst.SetCell(p, r, opts...); err != nil {
			return fmt.Errorf("dst.SetCell => %v", err)
		}
		return nil
	})

	// Neither of the two canvases (source and destination) have to be zero
	// based. Canvas is not zero based if it is positioned elsewhere, i.e.
	// providing a smaller view of another canvas.
	// E.g. a widget can assign a smaller portion of its canvas to a component
	// in order to restrict drawing of this component to a smaller area. To do
	// this it can create a sub-canvas. This sub-canvas can have a specific
	// starting position other than image.Point{0, 0} relative to the parent
	// canvas. Copying this sub-canvas back onto the parent accounts for this
	// offset.
	offset := c.area.Min
	return c.copyTo(offset, fn)
}
