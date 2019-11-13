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

// rectangle.go draws a rectangle.

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/canvas"
)

// RectangleOption is used to provide options to the Rectangle function.
type RectangleOption interface {
	// set sets the provided option.
	set(*rectOptions)
}

// rectOptions stores the provided options.
type rectOptions struct {
	cellOpts []cell.Option
	char     rune
}

// rectOption implements RectangleOption.
type rectOption func(rOpts *rectOptions)

// set implements RectangleOption.set.
func (ro rectOption) set(rOpts *rectOptions) {
	ro(rOpts)
}

// RectCellOpts sets options on the cells that create the rectangle.
func RectCellOpts(opts ...cell.Option) RectangleOption {
	return rectOption(func(rOpts *rectOptions) {
		rOpts.cellOpts = append(rOpts.cellOpts, opts...)
	})
}

// DefaultRectChar is the default value for the RectChar option.
const DefaultRectChar = ' '

// RectChar sets the character used in each of the cells of the rectangle.
func RectChar(c rune) RectangleOption {
	return rectOption(func(rOpts *rectOptions) {
		rOpts.char = c
	})
}

// Rectangle draws a filled rectangle on the canvas.
func Rectangle(c *canvas.Canvas, r image.Rectangle, opts ...RectangleOption) error {
	opt := &rectOptions{
		char: DefaultRectChar,
	}
	for _, o := range opts {
		o.set(opt)
	}

	if ar := c.Area(); !r.In(ar) {
		return fmt.Errorf("the requested rectangle %v doesn't fit the canvas area %v", r, ar)
	}

	if r.Dx() < 1 || r.Dy() < 1 {
		return fmt.Errorf("the rectangle must be at least 1x1 cell, got %v", r)
	}

	for col := r.Min.X; col < r.Max.X; col++ {
		for row := r.Min.Y; row < r.Max.Y; row++ {
			cells, err := c.SetCell(image.Point{col, row}, opt.char, opt.cellOpts...)
			if err != nil {
				return err
			}
			if cells != 1 {
				return fmt.Errorf("invalid rectangle character %q, this character occupies %d cells, the implementation only supports half-width runes that occupy exactly one cell", opt.char, cells)
			}
		}
	}
	return nil
}
