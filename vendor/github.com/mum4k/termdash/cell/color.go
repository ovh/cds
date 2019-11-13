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

package cell

import (
	"fmt"
)

// color.go defines constants for cell colors.

// Color is the color of a cell.
type Color int

// String implements fmt.Stringer()
func (cc Color) String() string {
	if n, ok := colorNames[cc]; ok {
		return n
	}
	return fmt.Sprintf("Color:%d", cc)
}

// colorNames maps Color values to human readable names.
var colorNames = map[Color]string{
	ColorDefault: "ColorDefault",
	ColorBlack:   "ColorBlack",
	ColorRed:     "ColorRed",
	ColorGreen:   "ColorGreen",
	ColorYellow:  "ColorYellow",
	ColorBlue:    "ColorBlue",
	ColorMagenta: "ColorMagenta",
	ColorCyan:    "ColorCyan",
	ColorWhite:   "ColorWhite",
}

// The supported terminal colors.
const (
	ColorDefault Color = iota

	// 8 "system" colors.
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

// ColorNumber sets a color using its number.
// Make sure your terminal is set to a terminalapi.ColorMode that supports the
// target color. The provided value must be in the range 0-255.
// Larger or smaller values will be reset to the default color.
//
// For reference on these colors see the Xterm number in:
// https://jonasjacek.github.io/colors/
func ColorNumber(n int) Color {
	if n < 0 || n > 255 {
		return ColorDefault
	}
	return Color(n + 1) // Colors are off-by-one due to ColorDefault being zero.
}

// ColorRGB6 sets a color using the 6x6x6 terminal color.
// Make sure your terminal is set to the terminalapi.ColorMode256 mode.
// The provided values (r, g, b) must be in the range 0-5.
// Larger or smaller values will be reset to the default color.
//
// For reference on these colors see:
// https://superuser.com/questions/783656/whats-the-deal-with-terminal-colors
func ColorRGB6(r, g, b int) Color {
	for _, c := range []int{r, g, b} {
		if c < 0 || c > 5 {
			return ColorDefault
		}
	}
	return Color(0x10 + 36*r + 6*g + b + 1) // Colors are off-by-one due to ColorDefault being zero.
}

// ColorRGB24 sets a color using the 24 bit web color scheme.
// Make sure your terminal is set to the terminalapi.ColorMode256 mode.
// The provided values (r, g, b) must be in the range 0-255.
// Larger or smaller values will be reset to the default color.
//
// For reference on these colors see the RGB column in:
// https://jonasjacek.github.io/colors/
func ColorRGB24(r, g, b int) Color {
	for _, c := range []int{r, g, b} {
		if c < 0 || c > 255 {
			return ColorDefault
		}
	}
	return ColorRGB6(r/51, g/51, b/51)
}
