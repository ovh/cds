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

// Package numbers implements various numerical functions.
package numbers

import (
	"image"
	"math"
)

// RoundToNonZeroPlaces rounds the float up, so that it has at least the provided
// number of non-zero decimal places.
// Returns the rounded float and the number of leading decimal places that
// are zero. Returns the original float when places is zero. Negative places
// are treated as positive, so that -2 == 2.
func RoundToNonZeroPlaces(f float64, places int) (float64, int) {
	if f == 0 {
		return 0, 0
	}

	decOnly := zeroBeforeDecimal(f)
	if decOnly == 0 {
		return f, 0
	}
	nzMult := multToNonZero(decOnly)
	if places == 0 {
		return f, multToPlaces(nzMult)
	}
	plMult := placesToMult(places)

	m := float64(nzMult * plMult)
	return math.Ceil(f*m) / m, multToPlaces(nzMult)
}

// multToNonZero returns multiplier for the float, so that the first decimal
// place is non-zero. The float must not be zero.
func multToNonZero(f float64) int {
	v := f
	if v < 0 {
		v *= -1
	}

	mult := 1
	for v < 0.1 {
		v *= 10
		mult *= 10
	}
	return mult
}

// placesToMult translates the number of decimal places to a multiple of 10.
func placesToMult(places int) int {
	if places < 0 {
		places *= -1
	}

	mult := 1
	for i := 0; i < places; i++ {
		mult *= 10
	}
	return mult
}

// multToPlaces translates the multiple of 10 to a number of decimal places.
func multToPlaces(mult int) int {
	places := 0
	for mult > 1 {
		mult /= 10
		places++
	}
	return places
}

// zeroBeforeDecimal modifies the float so that it only has zero value before
// the decimal point.
func zeroBeforeDecimal(f float64) float64 {
	var sign float64 = 1
	if f < 0 {
		f *= -1
		sign = -1
	}

	floor := math.Floor(f)
	return (f - floor) * sign
}

// MinMax returns the smallest and the largest value among the provided values.
// Returns (0, 0) if there are no values.
// Ignores NaN values. Allowing NaN values could lead to a corner case where all
// values can be NaN, in this case the function will return NaN as min and max.
func MinMax(values []float64) (min, max float64) {
	if len(values) == 0 {
		return 0, 0
	}
	min = math.MaxFloat64
	max = -1 * math.MaxFloat64
	allNaN := true
	for _, v := range values {
		if math.IsNaN(v) {
			continue
		}
		allNaN = false

		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if allNaN {
		return math.NaN(), math.NaN()
	}

	return min, max
}

// MinMaxInts returns the smallest and the largest int value among the provided
// values. Returns (0, 0) if there are no values.
func MinMaxInts(values []int) (min, max int) {
	if len(values) == 0 {
		return 0, 0
	}
	min = math.MaxInt32
	max = -1 * math.MaxInt32

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

// DegreesToRadians converts degrees to the equivalent in radians.
func DegreesToRadians(degrees int) float64 {
	if degrees > 360 {
		degrees %= 360
	}
	return (float64(degrees) / 180) * math.Pi
}

// RadiansToDegrees converts radians to the equivalent in degrees.
func RadiansToDegrees(radians float64) int {
	d := int(math.Round(radians * 180 / math.Pi))
	if d < 0 {
		d += 360
	}
	return d
}

// Abs returns the absolute value of x.
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// findGCF finds the greatest common factor of two integers.
func findGCF(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	a = Abs(a)
	b = Abs(b)

	// https://en.wikipedia.org/wiki/Euclidean_algorithm
	for {
		rem := a % b
		a = b
		b = rem

		if b == 0 {
			break
		}
	}
	return a
}

// SimplifyRatio simplifies the given ratio.
func SimplifyRatio(ratio image.Point) image.Point {
	gcf := findGCF(ratio.X, ratio.Y)
	if gcf == 0 {
		return image.ZP
	}
	return image.Point{
		X: ratio.X / gcf,
		Y: ratio.Y / gcf,
	}
}

// SplitByRatio splits the provided number by the specified ratio.
func SplitByRatio(n int, ratio image.Point) image.Point {
	sr := SimplifyRatio(ratio)
	if sr.Eq(image.ZP) {
		return image.ZP
	}
	fn := float64(n)
	sum := float64(sr.X + sr.Y)
	fact := fn / sum
	return image.Point{
		int(math.Round(fact * float64(sr.X))),
		int(math.Round(fact * float64(sr.Y))),
	}
}
