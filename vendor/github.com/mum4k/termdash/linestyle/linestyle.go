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

// Package linestyle defines various line styles.
package linestyle

// LineStyle defines the supported line styles.
type LineStyle int

// String implements fmt.Stringer()
func (ls LineStyle) String() string {
	if n, ok := lineStyleNames[ls]; ok {
		return n
	}
	return "LineStyleUnknown"
}

// lineStyleNames maps LineStyle values to human readable names.
var lineStyleNames = map[LineStyle]string{
	None:   "LineStyleNone",
	Light:  "LineStyleLight",
	Double: "LineStyleDouble",
	Round:  "LineStyleRound",
}

// Supported line styles.
// See https://en.wikipedia.org/wiki/Box-drawing_character.
const (
	// None indicates that no line should be present.
	None LineStyle = iota

	// Light is line style using the '─' characters.
	Light

	// Double is line style using the '═' characters.
	Double

	// Round is line style using the rounded corners '╭' characters.
	Round
)
