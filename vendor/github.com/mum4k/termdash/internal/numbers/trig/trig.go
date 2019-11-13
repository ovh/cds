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

// Package trig implements various trigonometrical calculations.
package trig

import (
	"fmt"
	"image"
	"math"
	"sort"

	"github.com/mum4k/termdash/internal/numbers"
)

// CirclePointAtAngle given an angle in degrees and a circle midpoint and
// radius, calculates coordinates of a point on the circle at that angle.
// Angles are zero at the X axis and grow counter-clockwise.
func CirclePointAtAngle(degrees int, mid image.Point, radius int) image.Point {
	angle := numbers.DegreesToRadians(degrees)
	r := float64(radius)
	x := mid.X + int(math.Round(r*math.Cos(angle)))
	// Y coordinates grow down on the canvas.
	y := mid.Y - int(math.Round(r*math.Sin(angle)))
	return image.Point{x, y}
}

// CircleAngleAtPoint given a point on a circle and its midpoint,
// calculates the angle in degrees.
// Angles are zero at the X axis and grow counter-clockwise.
func CircleAngleAtPoint(point, mid image.Point) int {
	adj := float64(point.X - mid.X)
	opp := float64(mid.Y - point.Y)
	if opp != 0 {
		angle := math.Atan2(opp, adj)
		return numbers.RadiansToDegrees(angle)
	} else if adj >= 0 {
		return 0
	} else {
		return 180
	}
}

// PointIsIn asserts whether the provided point is inside of a shape outlined
// with the provided points.
// Does not verify that the shape is closed or complete, it merely counts the
// number of intersections with the shape on one row.
func PointIsIn(p image.Point, points []image.Point) bool {
	maxX := p.X
	set := map[image.Point]struct{}{}
	for _, sp := range points {
		set[sp] = struct{}{}
		if sp.X > maxX {
			maxX = sp.X
		}
	}

	if _, ok := set[p]; ok {
		// Not inside if it is on the shape.
		return false
	}

	byY := map[int][]int{} // maps y->x
	for p := range set {
		byY[p.Y] = append(byY[p.Y], p.X)
	}
	for y := range byY {
		sort.Ints(byY[y])
	}

	set = map[image.Point]struct{}{}
	for y, xses := range byY {
		set[image.Point{xses[0], y}] = struct{}{}
		if len(xses) == 1 {
			continue
		}

		for i := 1; i < len(xses); i++ {
			if xses[i] != xses[i-1]+1 {
				set[image.Point{xses[i], y}] = struct{}{}
			}
		}
	}

	crosses := 0
	for x := p.X; x <= maxX; x++ {
		if _, ok := set[image.Point{x, p.Y}]; ok {
			crosses++
		}
	}
	return crosses%2 != 0
}

const (
	// MinAngle is the smallest valid angle in degrees.
	MinAngle = 0
	// MaxAngle is the largest valid angle in degrees.
	MaxAngle = 360
)

// angleRange represents a range of angles in degrees.
// The range includes all angles such that start <= angle <= end.
type angleRange struct {
	// start is the start if the range.
	// This is always less or equal to the end.
	start int

	// end is the end of the range.
	end int
}

// contains asserts whether the specified angle is in the range.
func (ar *angleRange) contains(angle int) bool {
	return angle >= ar.start && angle <= ar.end
}

// normalizeRange normalizes the start and end angles in degrees into ranges of
// angles. Useful for cases where the 0/360 point falls within the range.
// E.g:
//   0,25   => angleRange{0, 26}
//   0,360  => angleRange{0, 361}
//   359,20 => angleRange{359, 361}, angleRange{0, 21}
func normalizeRange(start, end int) ([]*angleRange, error) {
	if start < MinAngle || start > MaxAngle {
		return nil, fmt.Errorf("invalid start angle:%d, must be in range %d <= start <= %d", start, MinAngle, MaxAngle)
	}
	if end < MinAngle || end > MaxAngle {
		return nil, fmt.Errorf("invalid end angle:%d, must be in range %d <= end <= %d", end, MinAngle, MaxAngle)
	}

	if start == MaxAngle && end == 0 {
		start, end = end, start
	}

	if start <= end {
		return []*angleRange{
			{start, end},
		}, nil
	}

	// The range is crossing the 0/360 degree point.
	// Break it into multiple ranges.
	return []*angleRange{
		{start, MaxAngle},
		{0, end},
	}, nil
}

// RangeSize returns the size of the degree range.
// E.g:
//   0,25  => 25
//   359,1 => 2
func RangeSize(start, end int) (int, error) {
	ranges, err := normalizeRange(start, end)
	if err != nil {
		return 0, err
	}
	if len(ranges) == 1 {
		return end - start, nil
	}
	return MaxAngle - start + end, nil
}

// RangeMid returns an angle that lies in the middle between start and end.
// E.g:
//   0,10   => 5
//   350,10 => 0
func RangeMid(start, end int) (int, error) {
	ranges, err := normalizeRange(start, end)
	if err != nil {
		return 0, err
	}
	if len(ranges) == 1 {
		return start + ((end - start) / 2), nil
	}

	length := MaxAngle - start + end
	want := length / 2
	res := start + want
	return res % MaxAngle, nil
}

// FilterByAngle filters the provided points, returning only those that fall
// within the starting and the ending angle on a circle with the provided mid
// point.
func FilterByAngle(points []image.Point, mid image.Point, start, end int) ([]image.Point, error) {
	var res []image.Point
	ranges, err := normalizeRange(start, end)
	if err != nil {
		return nil, err
	}
	if mid.X < 0 || mid.Y < 0 {
		return nil, fmt.Errorf("the mid point %v cannot have negative coordinates", mid)
	}

	for _, p := range points {
		angle := CircleAngleAtPoint(p, mid)

		// Edge case, this might mean 0 or 360.
		// Decide based on where we are starting.
		if angle == 0 && start > 0 {
			angle = MaxAngle
		}

		for _, r := range ranges {
			if r.contains(angle) {
				res = append(res, p)
				break
			}
		}
	}
	return res, nil
}
