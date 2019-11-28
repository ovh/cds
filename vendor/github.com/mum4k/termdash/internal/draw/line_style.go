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

import (
	"fmt"

	"github.com/mum4k/termdash/internal/runewidth"
	"github.com/mum4k/termdash/linestyle"
)

// line_style.go contains the Unicode characters used for drawing lines of
// different styles.

// lineStyleChars maps the line styles to the corresponding component characters.
// Source: http://en.wikipedia.org/wiki/Box-drawing_character.
var lineStyleChars = map[linestyle.LineStyle]map[linePart]rune{
	linestyle.Light: {
		hLine:             '─',
		vLine:             '│',
		topLeftCorner:     '┌',
		topRightCorner:    '┐',
		bottomLeftCorner:  '└',
		bottomRightCorner: '┘',
		hAndUp:            '┴',
		hAndDown:          '┬',
		vAndLeft:          '┤',
		vAndRight:         '├',
		vAndH:             '┼',
	},
	linestyle.Double: {
		hLine:             '═',
		vLine:             '║',
		topLeftCorner:     '╔',
		topRightCorner:    '╗',
		bottomLeftCorner:  '╚',
		bottomRightCorner: '╝',
		hAndUp:            '╩',
		hAndDown:          '╦',
		vAndLeft:          '╣',
		vAndRight:         '╠',
		vAndH:             '╬',
	},
	linestyle.Round: {
		hLine:             '─',
		vLine:             '│',
		topLeftCorner:     '╭',
		topRightCorner:    '╮',
		bottomLeftCorner:  '╰',
		bottomRightCorner: '╯',
		hAndUp:            '┴',
		hAndDown:          '┬',
		vAndLeft:          '┤',
		vAndRight:         '├',
		vAndH:             '┼',
	},
}

// init verifies that all line parts are half-width runes (occupy only one
// cell).
func init() {
	for ls, parts := range lineStyleChars {
		for part, r := range parts {
			if got := runewidth.RuneWidth(r); got > 1 {
				panic(fmt.Errorf("line style %v line part %v is a rune %c with width %v, all parts must be half-width runes (width of one)", ls, part, r, got))
			}
		}
	}
}

// lineParts returns the line component characters for the provided line style.
func lineParts(ls linestyle.LineStyle) (map[linePart]rune, error) {
	parts, ok := lineStyleChars[ls]
	if !ok {
		return nil, fmt.Errorf("unsupported line style %d", ls)
	}
	return parts, nil
}

// linePart identifies individual line parts.
type linePart int

// String implements fmt.Stringer()
func (lp linePart) String() string {
	if n, ok := linePartNames[lp]; ok {
		return n
	}
	return "linePartUnknown"
}

// linePartNames maps linePart values to human readable names.
var linePartNames = map[linePart]string{
	vLine:             "linePartVLine",
	topLeftCorner:     "linePartTopLeftCorner",
	topRightCorner:    "linePartTopRightCorner",
	bottomLeftCorner:  "linePartBottomLeftCorner",
	bottomRightCorner: "linePartBottomRightCorner",
	hAndUp:            "linePartHAndUp",
	hAndDown:          "linePartHAndDown",
	vAndLeft:          "linePartVAndLeft",
	vAndRight:         "linePartVAndRight",
	vAndH:             "linePartVAndH",
}

const (
	hLine linePart = iota
	vLine
	topLeftCorner
	topRightCorner
	bottomLeftCorner
	bottomRightCorner
	hAndUp
	hAndDown
	vAndLeft
	vAndRight
	vAndH
)
