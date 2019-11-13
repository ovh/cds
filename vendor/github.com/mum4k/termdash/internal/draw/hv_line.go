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

// hv_line.go contains code that draws horizontal and vertical lines.

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/canvas"
	"github.com/mum4k/termdash/linestyle"
)

// HVLineOption is used to provide options to HVLine().
type HVLineOption interface {
	// set sets the provided option.
	set(*hVLineOptions)
}

// hVLineOptions stores the provided options.
type hVLineOptions struct {
	cellOpts  []cell.Option
	lineStyle linestyle.LineStyle
}

// newHVLineOptions returns a new hVLineOptions instance.
func newHVLineOptions() *hVLineOptions {
	return &hVLineOptions{
		lineStyle: DefaultLineStyle,
	}
}

// hVLineOption implements HVLineOption.
type hVLineOption func(*hVLineOptions)

// set implements HVLineOption.set.
func (o hVLineOption) set(opts *hVLineOptions) {
	o(opts)
}

// DefaultLineStyle is the default value for the HVLineStyle option.
const DefaultLineStyle = linestyle.Light

// HVLineStyle sets the style of the line.
// Defaults to DefaultLineStyle.
func HVLineStyle(ls linestyle.LineStyle) HVLineOption {
	return hVLineOption(func(opts *hVLineOptions) {
		opts.lineStyle = ls
	})
}

// HVLineCellOpts sets options on the cells that contain the line.
func HVLineCellOpts(cOpts ...cell.Option) HVLineOption {
	return hVLineOption(func(opts *hVLineOptions) {
		opts.cellOpts = cOpts
	})
}

// HVLine represents one horizontal or vertical line.
type HVLine struct {
	// Start is the cell where the line starts.
	Start image.Point
	// End is the cell where the line ends.
	End image.Point
}

// HVLines draws horizontal or vertical lines. Handles drawing of the correct
// characters for locations where any two lines cross (e.g. a corner, a T shape
// or a cross). Each line must be at least two cells long. Both start and end
// must be on the same horizontal (same X coordinate) or same vertical (same Y
// coordinate) line.
func HVLines(c *canvas.Canvas, lines []HVLine, opts ...HVLineOption) error {
	opt := newHVLineOptions()
	for _, o := range opts {
		o.set(opt)
	}

	g := newHVLineGraph()
	for _, l := range lines {
		line, err := newHVLine(c, l.Start, l.End, opt)
		if err != nil {
			return err
		}
		g.addLine(line)

		switch {
		case line.horizontal():
			for curX := line.start.X; ; curX++ {
				cur := image.Point{curX, line.start.Y}
				if _, err := c.SetCell(cur, line.mainPart, opt.cellOpts...); err != nil {
					return err
				}

				if curX == line.end.X {
					break
				}
			}

		case line.vertical():
			for curY := line.start.Y; ; curY++ {
				cur := image.Point{line.start.X, curY}
				if _, err := c.SetCell(cur, line.mainPart, opt.cellOpts...); err != nil {
					return err
				}

				if curY == line.end.Y {
					break
				}
			}
		}
	}

	for _, n := range g.multiEdgeNodes() {
		r, err := n.rune(opt.lineStyle)
		if err != nil {
			return err
		}
		if _, err := c.SetCell(n.p, r, opt.cellOpts...); err != nil {
			return err
		}
	}

	return nil
}

// hVLine represents a line that will be drawn on the canvas.
type hVLine struct {
	// start is the starting point of the line.
	start image.Point

	// end is the ending point of the line.
	end image.Point

	// mainPart is either parts[vLine] or parts[hLine] depending on whether
	// this is horizontal or vertical line.
	mainPart rune

	// opts are the options provided in a call to HVLine().
	opts *hVLineOptions
}

// newHVLine creates a new hVLine instance.
// Swaps start and end if necessary, so that horizontal drawing is always left
// to right and vertical is always top down.
func newHVLine(c *canvas.Canvas, start, end image.Point, opts *hVLineOptions) (*hVLine, error) {
	if ar := c.Area(); !start.In(ar) || !end.In(ar) {
		return nil, fmt.Errorf("both the start%v and the end%v must be in the canvas area: %v", start, end, ar)
	}

	parts, err := lineParts(opts.lineStyle)
	if err != nil {
		return nil, err
	}

	var mainPart rune
	switch {
	case start.X != end.X && start.Y != end.Y:
		return nil, fmt.Errorf("can only draw horizontal (same X coordinates) or vertical (same Y coordinates), got start:%v end:%v", start, end)

	case start.X == end.X && start.Y == end.Y:
		return nil, fmt.Errorf("the line must at least one cell long, got start%v, end%v", start, end)

	case start.X == end.X:
		mainPart = parts[vLine]
		if start.Y > end.Y {
			start, end = end, start
		}

	case start.Y == end.Y:
		mainPart = parts[hLine]
		if start.X > end.X {
			start, end = end, start
		}

	}

	return &hVLine{
		start:    start,
		end:      end,
		mainPart: mainPart,
		opts:     opts,
	}, nil
}

// horizontal determines if this is a horizontal line.
func (hvl *hVLine) horizontal() bool {
	return hvl.mainPart == lineStyleChars[hvl.opts.lineStyle][hLine]
}

// vertical determines if this is a vertical line.
func (hvl *hVLine) vertical() bool {
	return hvl.mainPart == lineStyleChars[hvl.opts.lineStyle][vLine]
}
