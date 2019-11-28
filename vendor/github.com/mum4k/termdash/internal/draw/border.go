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

// border.go contains code that draws borders.

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/align"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/alignfor"
	"github.com/mum4k/termdash/internal/canvas"
	"github.com/mum4k/termdash/linestyle"
)

// BorderOption is used to provide options to Border().
type BorderOption interface {
	// set sets the provided option.
	set(*borderOptions)
}

// borderOptions stores the provided options.
type borderOptions struct {
	cellOpts      []cell.Option
	lineStyle     linestyle.LineStyle
	title         string
	titleOM       OverrunMode
	titleCellOpts []cell.Option
	titleHAlign   align.Horizontal
}

// borderOption implements BorderOption.
type borderOption func(bOpts *borderOptions)

// set implements BorderOption.set.
func (bo borderOption) set(bOpts *borderOptions) {
	bo(bOpts)
}

// DefaultBorderLineStyle is the default value for the BorderLineStyle option.
const DefaultBorderLineStyle = linestyle.Light

// BorderLineStyle sets the style of the line used to draw the border.
func BorderLineStyle(ls linestyle.LineStyle) BorderOption {
	return borderOption(func(bOpts *borderOptions) {
		bOpts.lineStyle = ls
	})
}

// BorderCellOpts sets options on the cells that create the border.
func BorderCellOpts(opts ...cell.Option) BorderOption {
	return borderOption(func(bOpts *borderOptions) {
		bOpts.cellOpts = opts
	})
}

// BorderTitle sets a title for the border.
func BorderTitle(title string, overrun OverrunMode, opts ...cell.Option) BorderOption {
	return borderOption(func(bOpts *borderOptions) {
		bOpts.title = title
		bOpts.titleOM = overrun
		bOpts.titleCellOpts = opts
	})
}

// BorderTitleAlign configures the horizontal alignment for the title.
func BorderTitleAlign(h align.Horizontal) BorderOption {
	return borderOption(func(bOpts *borderOptions) {
		bOpts.titleHAlign = h
	})
}

// borderChar returns the correct border character from the parts for the use
// at the specified point of the border. Returns -1 if no character should be at
// this point.
func borderChar(p image.Point, border image.Rectangle, parts map[linePart]rune) rune {
	switch {
	case p.X == border.Min.X && p.Y == border.Min.Y:
		return parts[topLeftCorner]
	case p.X == border.Max.X-1 && p.Y == border.Min.Y:
		return parts[topRightCorner]
	case p.X == border.Min.X && p.Y == border.Max.Y-1:
		return parts[bottomLeftCorner]
	case p.X == border.Max.X-1 && p.Y == border.Max.Y-1:
		return parts[bottomRightCorner]
	case p.X == border.Min.X || p.X == border.Max.X-1:
		return parts[vLine]
	case p.Y == border.Min.Y || p.Y == border.Max.Y-1:
		return parts[hLine]
	}
	return -1
}

// drawTitle draws a text title at the top of the border.
func drawTitle(c *canvas.Canvas, border image.Rectangle, opt *borderOptions) error {
	// Don't attempt to draw the title if there isn't space for at least one rune.
	// The title must not overwrite any of the corner runes on the border so we
	// need the following minimum width.
	const minForTitle = 3
	if border.Dx() < minForTitle {
		return nil
	}

	available := image.Rect(
		border.Min.X+1, // One space for the top left corner char.
		border.Min.Y,
		border.Max.X-1, // One space for the top right corner char.
		border.Min.Y+1,
	)
	start, err := alignfor.Text(available, opt.title, opt.titleHAlign, align.VerticalTop)
	if err != nil {
		return err
	}

	return Text(
		c, opt.title, start,
		TextCellOpts(opt.titleCellOpts...),
		TextOverrunMode(opt.titleOM),
		TextMaxX(available.Max.X),
	)
}

// Border draws a border on the canvas.
func Border(c *canvas.Canvas, border image.Rectangle, opts ...BorderOption) error {
	if ar := c.Area(); !border.In(ar) {
		return fmt.Errorf("the requested border %+v falls outside of the provided canvas %+v", border, ar)
	}

	const minSize = 2
	if border.Dx() < minSize || border.Dy() < minSize {
		return fmt.Errorf("the smallest supported border is %dx%d, got: %dx%d", minSize, minSize, border.Dx(), border.Dy())
	}

	opt := &borderOptions{
		lineStyle: DefaultBorderLineStyle,
	}
	for _, o := range opts {
		o.set(opt)
	}

	parts, err := lineParts(opt.lineStyle)
	if err != nil {
		return err
	}

	for col := border.Min.X; col < border.Max.X; col++ {
		for row := border.Min.Y; row < border.Max.Y; row++ {
			p := image.Point{col, row}
			r := borderChar(p, border, parts)
			if r == -1 {
				continue
			}

			cells, err := c.SetCell(p, r, opt.cellOpts...)
			if err != nil {
				return err
			}
			if cells != 1 {
				panic(fmt.Sprintf("invalid border rune %q, this rune occupies %d cells, border implementation only supports half-width runes that occupy exactly one cell", r, cells))
			}
		}
	}

	if opt.title != "" {
		return drawTitle(c, border, opt)
	}
	return nil
}
