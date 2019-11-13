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

// Package buffer implements a 2-D buffer of cells.
package buffer

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/area"
	"github.com/mum4k/termdash/internal/runewidth"
)

// NewCells breaks the provided text into cells and applies the options.
func NewCells(text string, opts ...cell.Option) []*Cell {
	var res []*Cell
	for _, r := range text {
		res = append(res, NewCell(r, opts...))
	}
	return res
}

// Cell represents a single cell on the terminal.
type Cell struct {
	// Rune is the rune stored in the cell.
	Rune rune

	// Opts are the cell options.
	Opts *cell.Options
}

// String implements fmt.Stringer.
func (c *Cell) String() string {
	return fmt.Sprintf("{%q}", c.Rune)
}

// NewCell returns a new cell.
func NewCell(r rune, opts ...cell.Option) *Cell {
	return &Cell{
		Rune: r,
		Opts: cell.NewOptions(opts...),
	}
}

// Copy returns a copy the cell.
func (c *Cell) Copy() *Cell {
	return &Cell{
		Rune: c.Rune,
		Opts: cell.NewOptions(c.Opts),
	}
}

// Apply applies the provided options to the cell.
func (c *Cell) Apply(opts ...cell.Option) {
	for _, opt := range opts {
		opt.Set(c.Opts)
	}
}

// Buffer is a 2-D buffer of cells.
// The axes increase right and down.
// Uninitialized buffer is invalid, use New to create an instance.
// Don't set cells directly, use the SetCell method instead which safely
// handles limits and wide unicode characters.
type Buffer [][]*Cell

// New returns a new Buffer of the provided size.
func New(size image.Point) (Buffer, error) {
	if size.X <= 0 {
		return nil, fmt.Errorf("invalid buffer width (size.X): %d, must be a positive number", size.X)
	}
	if size.Y <= 0 {
		return nil, fmt.Errorf("invalid buffer height (size.Y): %d, must be a positive number", size.Y)
	}

	b := make([][]*Cell, size.X)
	for col := range b {
		b[col] = make([]*Cell, size.Y)
		for row := range b[col] {
			b[col][row] = NewCell(0)
		}
	}
	return b, nil
}

// SetCell sets the rune of the specified cell in the buffer. Returns the
// number of cells the rune occupies, wide runes can occupy multiple cells when
// printed on the terminal. See http://www.unicode.org/reports/tr11/.
// Use the options to specify which attributes to modify, if an attribute
// option isn't specified, the attribute retains its previous value.
func (b Buffer) SetCell(p image.Point, r rune, opts ...cell.Option) (int, error) {
	partial, err := b.IsPartial(p)
	if err != nil {
		return -1, err
	}
	if partial {
		return -1, fmt.Errorf("cannot set rune %q at point %v, it is a partial cell occupied by a wide rune in the previous cell", r, p)
	}

	remW, err := b.RemWidth(p)
	if err != nil {
		return -1, err
	}
	rw := runewidth.RuneWidth(r)
	if rw == 0 {
		// Even if the rune is invisible, like the zero-value rune, it still
		// occupies at least the target cell.
		rw = 1
	}
	if rw > remW {
		return -1, fmt.Errorf("cannot set rune %q of width %d at point %v, only have %d remaining cells at this line", r, rw, p, remW)
	}

	c := b[p.X][p.Y]
	c.Rune = r
	c.Apply(opts...)
	return rw, nil
}

// IsPartial returns true if the cell at the specified point holds a part of a
// full width rune from a previous cell. See
// http://www.unicode.org/reports/tr11/.
func (b Buffer) IsPartial(p image.Point) (bool, error) {
	size := b.Size()
	ar, err := area.FromSize(size)
	if err != nil {
		return false, err
	}

	if !p.In(ar) {
		return false, fmt.Errorf("point %v falls outside of the area %v occupied by the buffer", p, ar)
	}

	if p.X == 0 && p.Y == 0 {
		return false, nil
	}

	prevP := image.Point{p.X - 1, p.Y}
	if prevP.X < 0 {
		prevP = image.Point{size.X - 1, p.Y - 1}
	}

	prevR := b[prevP.X][prevP.Y].Rune
	switch rw := runewidth.RuneWidth(prevR); rw {
	case 0, 1:
		return false, nil
	case 2:
		return true, nil
	default:
		return false, fmt.Errorf("buffer cell %v contains rune %q which has an unsupported rune with %d", prevP, prevR, rw)
	}
}

// RemWidth returns the remaining width (horizontal row of cells) available
// from and inclusive of the specified point.
func (b Buffer) RemWidth(p image.Point) (int, error) {
	size := b.Size()
	ar, err := area.FromSize(size)
	if err != nil {
		return -1, err
	}

	if !p.In(ar) {
		return -1, fmt.Errorf("point %v falls outside of the area %v occupied by the buffer", p, ar)
	}
	return size.X - p.X, nil
}

// Size returns the size of the buffer.
func (b Buffer) Size() image.Point {
	return image.Point{
		len(b),
		len(b[0]),
	}
}
