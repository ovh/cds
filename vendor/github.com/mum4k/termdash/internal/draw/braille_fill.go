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

// braille_fill.go implements the flood-fill algorithm for filling shapes on the braille canvas.

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/canvas/braille"
)

// BrailleFillOption is used to provide options to BrailleFill.
type BrailleFillOption interface {
	// set sets the provided option.
	set(*brailleFillOptions)
}

// brailleFillOptions stores the provided options.
type brailleFillOptions struct {
	cellOpts    []cell.Option
	pixelChange braillePixelChange
}

// newBrailleFillOptions returns a new brailleFillOptions instance.
func newBrailleFillOptions() *brailleFillOptions {
	return &brailleFillOptions{
		pixelChange: braillePixelChangeSet,
	}
}

// brailleFillOption implements BrailleFillOption.
type brailleFillOption func(*brailleFillOptions)

// set implements BrailleFillOption.set.
func (o brailleFillOption) set(opts *brailleFillOptions) {
	o(opts)
}

// BrailleFillCellOpts sets options on the cells that are set as part of
// filling shapes.
// Cell options on a braille canvas can only be set on the entire cell, not per
// pixel.
func BrailleFillCellOpts(cOpts ...cell.Option) BrailleFillOption {
	return brailleFillOption(func(opts *brailleFillOptions) {
		opts.cellOpts = cOpts
	})
}

// BrailleFillClearPixels changes the behavior of BrailleFill, so that it
// clears the pixels instead of setting them.
// Useful in order to "erase" the filled area as opposed to drawing one.
func BrailleFillClearPixels() BrailleFillOption {
	return brailleFillOption(func(opts *brailleFillOptions) {
		opts.pixelChange = braillePixelChangeClear
	})
}

// BrailleFill fills the braille canvas starting at the specified point.
// The function will not fill or cross over any points in the defined border.
// The start point must be in the canvas.
func BrailleFill(bc *braille.Canvas, start image.Point, border []image.Point, opts ...BrailleFillOption) error {
	if ar := bc.Area(); !start.In(ar) {
		return fmt.Errorf("unable to start filling canvas at point %v which is outside of the braille canvas area %v", start, ar)
	}

	opt := newBrailleFillOptions()
	for _, o := range opts {
		o.set(opt)
	}

	b := map[image.Point]struct{}{}
	for _, p := range border {
		b[p] = struct{}{}
	}

	v := newVisitable(bc.Area(), b)
	visitor := func(p image.Point) error {
		switch opt.pixelChange {
		case braillePixelChangeSet:
			return bc.SetPixel(p, opt.cellOpts...)
		case braillePixelChangeClear:
			return bc.ClearPixel(p, opt.cellOpts...)
		}
		return nil
	}
	return brailleDFS(v, start, visitor)
}

// visitable represents an area that can be visited.
// It tracks nodes that are already visited.
type visitable struct {
	area    image.Rectangle
	visited map[image.Point]struct{}
}

// newVisitable returns a new visitable object initialized for the provided
// area and already visited nodes.
func newVisitable(ar image.Rectangle, visited map[image.Point]struct{}) *visitable {
	if visited == nil {
		visited = map[image.Point]struct{}{}
	}
	return &visitable{
		area:    ar,
		visited: visited,
	}
}

// neighborsAt returns all valid neighbors for the specified point.
func (v *visitable) neighborsAt(p image.Point) []image.Point {
	var res []image.Point
	for _, neigh := range []image.Point{
		{p.X - 1, p.Y}, // left
		{p.X + 1, p.Y}, // right
		{p.X, p.Y - 1}, // up
		{p.X, p.Y + 1}, // down
	} {
		if !neigh.In(v.area) {
			continue
		}
		if _, ok := v.visited[neigh]; ok {
			continue
		}
		v.visited[neigh] = struct{}{}
		res = append(res, neigh)
	}
	return res
}

// brailleDFS visits every point in the area and runs the visitor function.
func brailleDFS(v *visitable, p image.Point, visitFn func(image.Point) error) error {
	neigh := v.neighborsAt(p)
	if len(neigh) == 0 {
		return nil
	}

	for _, n := range neigh {
		if err := visitFn(n); err != nil {
			return err
		}
		if err := brailleDFS(v, n, visitFn); err != nil {
			return err
		}
	}
	return nil
}
