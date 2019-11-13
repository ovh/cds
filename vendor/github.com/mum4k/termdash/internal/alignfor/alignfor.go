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

// Package alignfor provides functions that align elements.
package alignfor

import (
	"fmt"
	"image"
	"strings"

	"github.com/mum4k/termdash/align"
	"github.com/mum4k/termdash/internal/runewidth"
	"github.com/mum4k/termdash/internal/wrap"
)

// hAlign aligns the given area in the rectangle horizontally.
func hAlign(rect image.Rectangle, ar image.Rectangle, h align.Horizontal) (image.Rectangle, error) {
	gap := rect.Dx() - ar.Dx()
	switch h {
	case align.HorizontalRight:
		// Use gap from above.
	case align.HorizontalCenter:
		gap /= 2
	case align.HorizontalLeft:
		gap = 0
	default:
		return image.ZR, fmt.Errorf("unsupported horizontal alignment %v", h)
	}

	return image.Rect(
		rect.Min.X+gap,
		ar.Min.Y,
		rect.Min.X+gap+ar.Dx(),
		ar.Max.Y,
	), nil
}

// vAlign aligns the given area in the rectangle vertically.
func vAlign(rect image.Rectangle, ar image.Rectangle, v align.Vertical) (image.Rectangle, error) {
	gap := rect.Dy() - ar.Dy()
	switch v {
	case align.VerticalBottom:
		// Use gap from above.
	case align.VerticalMiddle:
		gap /= 2
	case align.VerticalTop:
		gap = 0
	default:
		return image.ZR, fmt.Errorf("unsupported vertical alignment %v", v)
	}

	return image.Rect(
		ar.Min.X,
		rect.Min.Y+gap,
		ar.Max.X,
		rect.Min.Y+gap+ar.Dy(),
	), nil
}

// Rectangle aligns the area within the rectangle returning the
// aligned area. The area must fall within the rectangle.
func Rectangle(rect image.Rectangle, ar image.Rectangle, h align.Horizontal, v align.Vertical) (image.Rectangle, error) {
	if !ar.In(rect) {
		return image.ZR, fmt.Errorf("cannot align area %v inside rectangle %v, the area falls outside of the rectangle", ar, rect)
	}

	aligned, err := hAlign(rect, ar, h)
	if err != nil {
		return image.ZR, err
	}
	aligned, err = vAlign(rect, aligned, v)
	if err != nil {
		return image.ZR, err
	}
	return aligned, nil
}

// Text aligns the text within the given rectangle, returns the start point for the text.
// For the purposes of the alignment this assumes that text will be trimmed if
// it overruns the rectangle.
// This only supports a single line of text, the text must not contain non-printable characters,
// allows empty text.
func Text(rect image.Rectangle, text string, h align.Horizontal, v align.Vertical) (image.Point, error) {
	if strings.ContainsRune(text, '\n') {
		return image.ZP, fmt.Errorf("the provided text contains a newline character: %q", text)
	}

	if text != "" {
		if err := wrap.ValidText(text); err != nil {
			return image.ZP, fmt.Errorf("the provided text contains non printable character(s): %s", err)
		}
	}

	cells := runewidth.StringWidth(text)
	var textLen int
	if cells < rect.Dx() {
		textLen = cells
	} else {
		textLen = rect.Dx()
	}

	textRect := image.Rect(
		rect.Min.X,
		rect.Min.Y,
		// For the purposes of aligning the text, assume that it will be
		// trimmed to the available space.
		rect.Min.X+textLen,
		rect.Min.Y+1,
	)

	aligned, err := Rectangle(rect, textRect, h, v)
	if err != nil {
		return image.ZP, err
	}
	return image.Point{aligned.Min.X, aligned.Min.Y}, nil
}
