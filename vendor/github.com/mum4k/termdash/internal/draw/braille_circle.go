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

// braille_circle.go contains code that draws circles on a braille canvas.

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/internal/canvas/braille"
	"github.com/mum4k/termdash/internal/numbers/trig"
)

// BrailleCircleOption is used to provide options to BrailleCircle.
type BrailleCircleOption interface {
	// set sets the provided option.
	set(*brailleCircleOptions)
}

// brailleCircleOptions stores the provided options.
type brailleCircleOptions struct {
	cellOpts    []cell.Option
	filled      bool
	pixelChange braillePixelChange

	arcOnly     bool
	startDegree int
	endDegree   int
}

// newBrailleCircleOptions returns a new brailleCircleOptions instance.
func newBrailleCircleOptions() *brailleCircleOptions {
	return &brailleCircleOptions{
		pixelChange: braillePixelChangeSet,
	}
}

// validate validates the provided options.
func (opts *brailleCircleOptions) validate() error {
	if !opts.arcOnly {
		return nil
	}

	if opts.startDegree == opts.endDegree {
		return fmt.Errorf("invalid degree range, start %d and end %d cannot be equal", opts.startDegree, opts.endDegree)
	}
	return nil
}

// brailleCircleOption implements BrailleCircleOption.
type brailleCircleOption func(*brailleCircleOptions)

// set implements BrailleCircleOption.set.
func (o brailleCircleOption) set(opts *brailleCircleOptions) {
	o(opts)
}

// BrailleCircleCellOpts sets options on the cells that contain the circle.
// Cell options on a braille canvas can only be set on the entire cell, not per
// pixel.
func BrailleCircleCellOpts(cOpts ...cell.Option) BrailleCircleOption {
	return brailleCircleOption(func(opts *brailleCircleOptions) {
		opts.cellOpts = cOpts
	})
}

// BrailleCircleFilled indicates that the drawn circle should be filled.
func BrailleCircleFilled() BrailleCircleOption {
	return brailleCircleOption(func(opts *brailleCircleOptions) {
		opts.filled = true
	})
}

// BrailleCircleArcOnly indicates that only a portion of the circle should be drawn.
// The arc will be between the two provided angles in degrees.
// Each angle must be in range 0 <= angle <= 360. Start and end must not be equal.
// The zero angle is on the X axis, angles grow counter-clockwise.
func BrailleCircleArcOnly(startDegree, endDegree int) BrailleCircleOption {
	return brailleCircleOption(func(opts *brailleCircleOptions) {
		opts.arcOnly = true
		opts.startDegree = startDegree
		opts.endDegree = endDegree

	})
}

// BrailleCircleClearPixels changes the behavior of BrailleCircle, so that it
// clears the pixels belonging to the circle instead of setting them.
// Useful in order to "erase" a circle from the canvas as opposed to drawing one.
func BrailleCircleClearPixels() BrailleCircleOption {
	return brailleCircleOption(func(opts *brailleCircleOptions) {
		opts.pixelChange = braillePixelChangeClear
	})
}

// BrailleCircle draws an approximated circle with the specified mid point and radius.
// The mid point must be a valid pixel within the canvas.
// All the points that form the circle must fit into the canvas.
// The smallest valid radius is two.
func BrailleCircle(bc *braille.Canvas, mid image.Point, radius int, opts ...BrailleCircleOption) error {
	if ar := bc.Area(); !mid.In(ar) {
		return fmt.Errorf("unable to draw circle with mid point %v which is outside of the braille canvas area %v", mid, ar)
	}
	if min := 2; radius < min {
		return fmt.Errorf("unable to draw circle with radius %d, must be in range %d <= radius", radius, min)
	}

	opt := newBrailleCircleOptions()
	for _, o := range opts {
		o.set(opt)
	}

	if err := opt.validate(); err != nil {
		return err
	}

	points := circlePoints(mid, radius)
	if opt.arcOnly {
		f, err := trig.FilterByAngle(points, mid, opt.startDegree, opt.endDegree)
		if err != nil {
			return err
		}
		points = f
		if opt.filled && (opt.startDegree != 0 || opt.endDegree != 360) {
			points = append(points, openingPoints(mid, radius, opt)...)
		}
	}
	if err := drawPoints(bc, points, opt); err != nil {
		return fmt.Errorf("failed to draw circle with mid:%v, radius:%d, start:%d degrees, end:%d degrees: %v", mid, radius, opt.startDegree, opt.endDegree, err)
	}
	if opt.filled {
		return fillCircle(bc, points, mid, radius, opt)
	}
	return nil
}

// drawPoints draws the points onto the canvas.
func drawPoints(bc *braille.Canvas, points []image.Point, opt *brailleCircleOptions) error {
	for _, p := range points {
		switch opt.pixelChange {
		case braillePixelChangeSet:
			if err := bc.SetPixel(p, opt.cellOpts...); err != nil {
				return fmt.Errorf("SetPixel => %v", err)
			}
		case braillePixelChangeClear:
			if err := bc.ClearPixel(p, opt.cellOpts...); err != nil {
				return fmt.Errorf("ClearPixel => %v", err)
			}

		}
	}
	return nil
}

// fillCircle fills a circle that consists of the provided point and has the
// mid point and radius.
func fillCircle(bc *braille.Canvas, points []image.Point, mid image.Point, radius int, opt *brailleCircleOptions) error {
	lineOpts := []BrailleLineOption{
		BrailleLineCellOpts(opt.cellOpts...),
	}
	fillOpts := []BrailleFillOption{
		BrailleFillCellOpts(opt.cellOpts...),
	}
	if opt.pixelChange == braillePixelChangeClear {
		lineOpts = append(lineOpts, BrailleLineClearPixels())
		fillOpts = append(fillOpts, BrailleFillClearPixels())
	}

	// Determine a fill point that should be inside of the circle sector.
	midA, err := trig.RangeMid(opt.startDegree, opt.endDegree)
	if err != nil {
		return err
	}
	fp := trig.CirclePointAtAngle(midA, mid, radius-1)

	// Ensure the fill point falls inside the circle.
	// If drawing a partial circle, it must also fall within points belonging
	// to the opening.
	// This might not be true if drawing a partial circle and the arc is very
	// small.
	shape := points
	if opt.arcOnly {
		startP := trig.CirclePointAtAngle(opt.startDegree, mid, radius-1)
		endP := trig.CirclePointAtAngle(opt.endDegree, mid, radius-1)
		shape = append(shape, startP, endP)
	}
	if trig.PointIsIn(fp, shape) {
		if err := BrailleFill(bc, fp, points, fillOpts...); err != nil {
			return err
		}
		if err := BrailleLine(bc, mid, fp, lineOpts...); err != nil {
			return err
		}
	}
	return nil
}

// openingPoints returns points on the lines from the mid point to the circle
// opening when drawing an incomplete circle.
func openingPoints(mid image.Point, radius int, opt *brailleCircleOptions) []image.Point {
	var points []image.Point
	startP := trig.CirclePointAtAngle(opt.startDegree, mid, radius)
	endP := trig.CirclePointAtAngle(opt.endDegree, mid, radius)
	points = append(points, brailleLinePoints(mid, startP)...)
	points = append(points, brailleLinePoints(mid, endP)...)
	return points
}

// circlePoints returns a list of points that represent a circle with
// the specified mid point and radius.
func circlePoints(mid image.Point, radius int) []image.Point {
	var points []image.Point

	// Bresenham algorithm.
	// https://en.wikipedia.org/wiki/Midpoint_circle_algorithm
	x := radius
	y := 0
	dx := 1
	dy := 1
	diff := dx - (radius << 1) // Cheap multiplication by two.

	for x >= y {
		points = append(
			points,
			image.Point{mid.X + x, mid.Y + y},
			image.Point{mid.X + y, mid.Y + x},
			image.Point{mid.X - y, mid.Y + x},
			image.Point{mid.X - x, mid.Y + y},
			image.Point{mid.X - x, mid.Y - y},
			image.Point{mid.X - y, mid.Y - x},
			image.Point{mid.X + y, mid.Y - x},
			image.Point{mid.X + x, mid.Y - y},
		)

		if diff <= 0 {
			y++
			diff += dy
			dy += 2
		}

		if diff > 0 {
			x--
			dx += 2
			diff += dx - (radius << 1)
		}

	}
	return points
}
