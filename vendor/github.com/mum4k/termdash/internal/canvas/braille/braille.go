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

/*
Package braille provides a canvas that uses braille characters.

This is inspired by https://github.com/asciimoo/drawille.

The braille patterns documentation:
http://www.alanwood.net/unicode/braille_patterns.html

The use of braille characters gives additional points (higher resolution) on
the canvas, each character cell now has eight pixels that can be set
independently. Specifically each cell has the following pixels, the axes grow
right and down.

Each cell:

  X→ 0 1  Y
    ┌───┐ ↓
    │● ●│ 0
    │● ●│ 1
    │● ●│ 2
    │● ●│ 3
    └───┘

When using the braille canvas, the coordinates address the sub-cell points
rather then cells themselves. However all points in the cell still share the
same cell options.
*/
package braille

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/canvas"
	"github.com/mum4k/termdash/terminal/terminalapi"
)

const (
	// ColMult is the resolution multiplier for the width, i.e. two pixels per cell.
	ColMult = 2

	// RowMult is the resolution multiplier for the height, i.e. four pixels per cell.
	RowMult = 4

	// brailleCharOffset is the offset of the braille pattern unicode characters.
	// From: http://www.alanwood.net/unicode/braille_patterns.html
	brailleCharOffset = 0x2800

	// brailleLastChar is the last braille pattern rune.
	brailleLastChar = 0x28FF
)

// pixelRunes maps points addressing individual pixels in a cell into character
// offset. I.e. the correct character to set pixel(0,0) is
// brailleCharOffset|pixelRunes[image.Point{0,0}].
var pixelRunes = map[image.Point]rune{
	{0, 0}: 0x01, {1, 0}: 0x08,
	{0, 1}: 0x02, {1, 1}: 0x10,
	{0, 2}: 0x04, {1, 2}: 0x20,
	{0, 3}: 0x40, {1, 3}: 0x80,
}

// Canvas is a canvas that uses the braille patterns. It is two times wider
// and four times taller than a regular canvas that uses just plain characters,
// since each cell now has 2x4 pixels that can be independently set.
//
// The braille canvas is an abstraction built on top of a regular character
// canvas. After setting and toggling pixels on the braille canvas, it should
// be copied to a regular character canvas or applied to a terminal which
// results in setting of braille pattern characters.
// See the examples for more details.
//
// The created braille canvas can be smaller and even misaligned relatively to
// the regular character canvas or terminal, allowing the callers to create a
// "view" of just a portion of the canvas or terminal.
type Canvas struct {
	// regular is the regular character canvas the braille canvas is based on.
	regular *canvas.Canvas
}

// New returns a new braille canvas for the provided area.
func New(ar image.Rectangle) (*Canvas, error) {
	rc, err := canvas.New(ar)
	if err != nil {
		return nil, err
	}
	return &Canvas{
		regular: rc,
	}, nil
}

// Size returns the size of the braille canvas in pixels.
func (c *Canvas) Size() image.Point {
	s := c.regular.Size()
	return image.Point{s.X * ColMult, s.Y * RowMult}
}

// CellArea returns the area of the underlying cell canvas in cells.
func (c *Canvas) CellArea() image.Rectangle {
	return c.regular.Area()
}

// Area returns the area of the braille canvas in pixels.
// This will be zero-based area that is two times wider and four times taller
// than the area used to create the braille canvas.
func (c *Canvas) Area() image.Rectangle {
	ar := c.regular.Area()
	return image.Rect(0, 0, ar.Dx()*ColMult, ar.Dy()*RowMult)
}

// Clear clears all the content on the canvas.
func (c *Canvas) Clear() error {
	return c.regular.Clear()
}

// SetPixel turns on pixel at the specified point.
// The provided cell options will be applied to the entire cell (all of its
// pixels). This method is idempotent.
func (c *Canvas) SetPixel(p image.Point, opts ...cell.Option) error {
	cp, err := c.cellPoint(p)
	if err != nil {
		return err
	}
	cell, err := c.regular.Cell(cp)
	if err != nil {
		return err
	}

	var r rune
	if isBraille(cell.Rune) {
		// If the cell already has a braille pattern rune, we will be adding
		// the pixel.
		r = cell.Rune
	} else {
		r = brailleCharOffset
	}

	r |= pixelRunes[pixelPoint(p)]
	if _, err := c.regular.SetCell(cp, r, opts...); err != nil {
		return err
	}
	return nil
}

// ClearPixel turns off pixel at the specified point.
// The provided cell options will be applied to the entire cell (all of its
// pixels). This method is idempotent.
func (c *Canvas) ClearPixel(p image.Point, opts ...cell.Option) error {
	cp, err := c.cellPoint(p)
	if err != nil {
		return err
	}
	cell, err := c.regular.Cell(cp)
	if err != nil {
		return err
	}

	// Clear is idempotent.
	if !isBraille(cell.Rune) || !pixelSet(cell.Rune, p) {
		return nil
	}

	r := cell.Rune & ^pixelRunes[pixelPoint(p)]
	if _, err := c.regular.SetCell(cp, r, opts...); err != nil {
		return err
	}
	return nil
}

// TogglePixel toggles the state of the pixel at the specified point, i.e. it
// either sets or clear it depending on its current state.
// The provided cell options will be applied to the entire cell (all of its
// pixels).
func (c *Canvas) TogglePixel(p image.Point, opts ...cell.Option) error {
	cp, err := c.cellPoint(p)
	if err != nil {
		return err
	}
	curCell, err := c.regular.Cell(cp)
	if err != nil {
		return err
	}

	if isBraille(curCell.Rune) && pixelSet(curCell.Rune, p) {
		return c.ClearPixel(p, opts...)
	}
	return c.SetPixel(p, opts...)
}

// SetCellOpts sets options on the specified cell of the braille canvas without
// modifying the content of the cell.
// Sets the default cell options if no options are provided.
// This method is idempotent.
func (c *Canvas) SetCellOpts(cellPoint image.Point, opts ...cell.Option) error {
	curCell, err := c.regular.Cell(cellPoint)
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
	if _, err := c.regular.SetCell(cellPoint, curCell.Rune, opts...); err != nil {
		return err
	}
	return nil
}

// SetAreaCellOpts is like SetCellOpts, but sets the specified options on all
// the cells within the provided area.
func (c *Canvas) SetAreaCellOpts(cellArea image.Rectangle, opts ...cell.Option) error {
	haveArea := c.regular.Area()
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

// Apply applies the canvas to the corresponding area of the terminal.
// Guarantees to stay within limits of the area the canvas was created with.
func (c *Canvas) Apply(t terminalapi.Terminal) error {
	return c.regular.Apply(t)
}

// CopyTo copies the content of this canvas onto the destination canvas.
// This canvas can have an offset when compared to the destination canvas, i.e.
// the area of this canvas doesn't have to be zero-based.
func (c *Canvas) CopyTo(dst *canvas.Canvas) error {
	return c.regular.CopyTo(dst)
}

// cellPoint determines the point (coordinate) of the character cell given
// coordinates in pixels.
func (c *Canvas) cellPoint(p image.Point) (image.Point, error) {
	if p.X < 0 || p.Y < 0 {
		return image.ZP, fmt.Errorf("pixels cannot have negative coordinates: %v", p)
	}
	cp := image.Point{p.X / ColMult, p.Y / RowMult}
	if ar := c.regular.Area(); !cp.In(ar) {
		return image.ZP, fmt.Errorf("pixel at%v would be in a character cell at%v which falls outside of the canvas area %v", p, cp, ar)
	}
	return cp, nil
}

// isBraille determines if the rune is a braille pattern rune.
func isBraille(r rune) bool {
	return r >= brailleCharOffset && r <= brailleLastChar
}

// pixelSet returns true if the provided rune has the specified pixel set.
func pixelSet(r rune, p image.Point) bool {
	return r&pixelRunes[pixelPoint(p)] > 0
}

// pixelPoint translates point within canvas to point within the target cell.
func pixelPoint(p image.Point) image.Point {
	return image.Point{p.X % ColMult, p.Y % RowMult}
}
