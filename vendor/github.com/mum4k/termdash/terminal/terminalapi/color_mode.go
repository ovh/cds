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

package terminalapi

// color_mode.go defines the terminal color modes.

// ColorMode represents a color mode of a terminal.
type ColorMode int

// String implements fmt.Stringer()
func (cm ColorMode) String() string {
	if n, ok := colorModeNames[cm]; ok {
		return n
	}
	return "ColorModeUnknown"
}

// colorModeNames maps ColorMode values to human readable names.
var colorModeNames = map[ColorMode]string{
	ColorModeNormal:    "ColorModeNormal",
	ColorMode256:       "ColorMode256",
	ColorMode216:       "ColorMode216",
	ColorModeGrayscale: "ColorModeGrayscale",
}

// Supported color modes.
const (
	// ColorModeNormal supports 8 "system" colors.
	// These are defined as constants in the cell package.
	ColorModeNormal ColorMode = iota

	// ColorMode256 enables using any of the 256 terminal colors.
	//     0-7: the 8 "system" colors accessible in ColorModeNormal.
	//    8-15: the 8 "bright system" colors.
	//  16-231: the 216 different terminal colors.
	// 232-255: the 24 different shades of grey.
	ColorMode256

	// ColorMode216 supports only the third range of the ColorMode256, i.e the
	// 216 different terminal colors. However in this mode the colors are zero
	// based, so the caller doesn't need to provide an offset.
	ColorMode216

	// ColorModeGrayscale supports only the fourth range of the ColorMode256,
	// i.e the 24 different shades of grey. However in this mode the colors are
	// zero based, so the caller doesn't need to provide an offset.
	ColorModeGrayscale
)
