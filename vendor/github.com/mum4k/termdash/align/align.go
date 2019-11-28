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

// Package align defines constants representing types of alignment.
package align

// Horizontal indicates the type of horizontal alignment.
type Horizontal int

// String implements fmt.Stringer()
func (h Horizontal) String() string {
	if n, ok := horizontalNames[h]; ok {
		return n
	}
	return "HorizontalUnknown"
}

// horizontalNames maps Horizontal values to human readable names.
var horizontalNames = map[Horizontal]string{
	HorizontalLeft:   "HorizontalLeft",
	HorizontalCenter: "HorizontalCenter",
	HorizontalRight:  "HorizontalRight",
}

const (
	// HorizontalLeft is left alignment along the horizontal axis.
	HorizontalLeft Horizontal = iota
	// HorizontalCenter is center alignment along the horizontal axis.
	HorizontalCenter
	// HorizontalRight is right alignment along the horizontal axis.
	HorizontalRight
)

// Vertical indicates the type of vertical alignment.
type Vertical int

// String implements fmt.Stringer()
func (v Vertical) String() string {
	if n, ok := verticalNames[v]; ok {
		return n
	}
	return "VerticalUnknown"
}

// verticalNames maps Vertical values to human readable names.
var verticalNames = map[Vertical]string{
	VerticalTop:    "VerticalTop",
	VerticalMiddle: "VerticalMiddle",
	VerticalBottom: "VerticalBottom",
}

const (
	// VerticalTop is top alignment along the vertical axis.
	VerticalTop Vertical = iota
	// VerticalMiddle is middle alignment along the vertical axis.
	VerticalMiddle
	// VerticalBottom is bottom alignment along the vertical axis.
	VerticalBottom
)
