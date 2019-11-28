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

// Package wrap implements line wrapping at character or word boundaries.
package wrap

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/mum4k/termdash/internal/canvas/buffer"
	"github.com/mum4k/termdash/internal/runewidth"
)

// Mode sets the wrapping mode.
type Mode int

// String implements fmt.Stringer()
func (m Mode) String() string {
	if n, ok := modeNames[m]; ok {
		return n
	}
	return "ModeUnknown"
}

// modeNames maps Mode values to human readable names.
var modeNames = map[Mode]string{
	Never:   "WrapModeNever",
	AtRunes: "WrapModeAtRunes",
	AtWords: "WrapModeAtWords",
}

const (
	// Never is the default wrapping mode, which disables line wrapping.
	Never Mode = iota

	// AtRunes is a wrapping mode where if the width of the text crosses the
	// width of the canvas, wrapping is performed at rune boundaries.
	AtRunes

	// AtWords is a wrapping mode where if the width of the text crosses the
	// width of the canvas, wrapping is performed at word boundaries. The
	// wrapping still switches back to the AtRunes mode for any words that are
	// longer than the width.
	AtWords
)

// ValidText validates the provided text for wrapping.
// The text must not be empty, contain any control or
// space characters other than '\n' and ' '.
func ValidText(text string) error {
	if text == "" {
		return errors.New("the text cannot be empty")
	}

	for _, c := range text {
		if c == ' ' || c == '\n' { // Allowed space and control runes.
			continue
		}
		if unicode.IsControl(c) {
			return fmt.Errorf("the provided text %q cannot contain control characters, found: %q", text, c)
		}
		if unicode.IsSpace(c) {
			return fmt.Errorf("the provided text %q cannot contain space character %q", text, c)
		}
	}
	return nil
}

// ValidCells validates the provided cells for wrapping.
// The text in the cells must follow the same rules as described for ValidText.
func ValidCells(cells []*buffer.Cell) error {
	var b strings.Builder
	for _, c := range cells {
		b.WriteRune(c.Rune)
	}
	return ValidText(b.String())
}

// Cells returns the cells wrapped into individual lines according to the
// specified width and wrapping mode.
//
// This function consumes any cells that contain newline characters and uses
// them to start new lines.
//
// If the mode is AtWords, this function also drops cells with leading space
// character before a word at which the wrap occurs.
func Cells(cells []*buffer.Cell, width int, m Mode) ([][]*buffer.Cell, error) {
	if err := ValidCells(cells); err != nil {
		return nil, err
	}
	switch m {
	case Never:
	case AtRunes:
	case AtWords:
	default:
		return nil, fmt.Errorf("unsupported wrapping mode %v(%d)", m, m)
	}
	if width <= 0 {
		return nil, nil
	}

	cs := newCellScanner(cells, width, m)
	for state := scanCellRunes; state != nil; state = state(cs) {
	}
	return cs.lines, nil
}

// cellScannerState is a state in the FSM that scans the input text and identifies
// newlines.
type cellScannerState func(*cellScanner) cellScannerState

// cellScanner tracks the progress of scanning the input cells when finding
// lines.
type cellScanner struct {
	// cells are the cells being scanned.
	cells []*buffer.Cell

	// nextIdx is the index of the cell that will be returned by next.
	nextIdx int

	// wordStartIdx stores the starting index of the current word.
	// A starting position of a word includes any leading space characters.
	// E.g.: hello   world
	//            ^
	//            lastWordIdx
	wordStartIdx int
	// wordEndIdx stores the ending index of the current word.
	// The word consists of all indexes that are
	// wordStartIdx <= idx < wordEndIdx.
	// A word also includes any punctuation after it.
	wordEndIdx int

	// width is the width of the canvas the text will be drawn on.
	width int

	// posX tracks the horizontal position of the current cell on the canvas.
	posX int

	// mode is the wrapping mode.
	mode Mode

	// atRunesInWord overrides the mode back to AtRunes.
	atRunesInWord bool

	// lines are the identified lines.
	lines [][]*buffer.Cell

	// line is the current line.
	line []*buffer.Cell
}

// newCellScanner returns a scanner of the provided cells.
func newCellScanner(cells []*buffer.Cell, width int, m Mode) *cellScanner {
	return &cellScanner{
		cells: cells,
		width: width,
		mode:  m,
	}
}

// next returns the next cell and advances the scanner.
// Returns nil when there are no more cells to scan.
func (cs *cellScanner) next() *buffer.Cell {
	c := cs.peek()
	if c != nil {
		cs.nextIdx++
	}
	return c
}

// peek returns the next cell without advancing the scanner's position.
// Returns nil when there are no more cells to peek at.
func (cs *cellScanner) peek() *buffer.Cell {
	if cs.nextIdx >= len(cs.cells) {
		return nil
	}
	return cs.cells[cs.nextIdx]
}

// peekPrev returns the previous cell without changing the scanner's position.
// Returns nil if the scanner is at the first cell.
func (cs *cellScanner) peekPrev() *buffer.Cell {
	if cs.nextIdx == 0 {
		return nil
	}
	return cs.cells[cs.nextIdx-1]
}

// wordCells returns all the cells that belong to the current word.
func (cs *cellScanner) wordCells() []*buffer.Cell {
	return cs.cells[cs.wordStartIdx:cs.wordEndIdx]
}

// wordWidth returns the width of the current word in cells when printed on the
// terminal.
func (cs *cellScanner) wordWidth() int {
	var b strings.Builder
	for _, wc := range cs.wordCells() {
		b.WriteRune(wc.Rune)
	}
	return runewidth.StringWidth(b.String())
}

// isWordStart determines if the scanner is at the beginning of a word.
func (cs *cellScanner) isWordStart() bool {
	if cs.mode != AtWords {
		return false
	}

	current := cs.peekPrev()
	next := cs.peek()
	if current == nil || next == nil {
		return false
	}

	switch nr := next.Rune; {
	case nr == '\n':
	case nr == ' ':
	default:
		return true
	}
	return false
}

// scanCellRunes scans the cells a rune at a time.
func scanCellRunes(cs *cellScanner) cellScannerState {
	for {
		cell := cs.next()
		if cell == nil {
			return scanEOF
		}

		r := cell.Rune
		if r == '\n' {
			return newLineForLineBreak
		}

		if cs.mode == Never {
			return runeToCurrentLine
		}

		if cs.atRunesInWord && !isWordCell(cell) {
			cs.atRunesInWord = false
		}

		if !cs.atRunesInWord && cs.isWordStart() {
			return markWordStart
		}

		if runeWrapNeeded(r, cs.posX, cs.width) {
			return newLineForAtRunes
		}

		return runeToCurrentLine
	}
}

// runeToCurrentLine scans a single cell rune onto the current line.
func runeToCurrentLine(cs *cellScanner) cellScannerState {
	cell := cs.peekPrev()
	// Move horizontally within the line for each scanned cell.
	cs.posX += runewidth.RuneWidth(cell.Rune)

	// Copy the cell into the current line.
	cs.line = append(cs.line, cell)
	return scanCellRunes
}

// newLineForLineBreak processes a newline character cell.
func newLineForLineBreak(cs *cellScanner) cellScannerState {
	cs.lines = append(cs.lines, cs.line)
	cs.posX = 0
	cs.line = nil
	return scanCellRunes
}

// newLineForAtRunes processes a line wrap at rune boundaries due to canvas width.
func newLineForAtRunes(cs *cellScanner) cellScannerState {
	// The character on which we wrapped will be printed and is the start of
	// new line.
	cs.lines = append(cs.lines, cs.line)
	cs.posX = runewidth.RuneWidth(cs.peekPrev().Rune)
	cs.line = []*buffer.Cell{cs.peekPrev()}
	return scanCellRunes
}

// scanEOF terminates the scanning.
func scanEOF(cs *cellScanner) cellScannerState {
	// Need to add the current line if it isn't empty, or if the previous rune
	// was a newline.
	// Newlines aren't copied onto the lines so just checking for emptiness
	// isn't enough. We still want to include trailing empty newlines if
	// they are in the input text.
	if len(cs.line) > 0 || cs.peekPrev().Rune == '\n' {
		cs.lines = append(cs.lines, cs.line)
	}
	return nil
}

// markWordStart stores the starting position of the current word.
func markWordStart(cs *cellScanner) cellScannerState {
	cs.wordStartIdx = cs.nextIdx - 1
	cs.wordEndIdx = cs.nextIdx
	return scanWord
}

// scanWord scans the entire word until it finds its end.
func scanWord(cs *cellScanner) cellScannerState {
	for {
		if isWordCell(cs.peek()) {
			cs.next()
			cs.wordEndIdx++
			continue
		}
		return wordToCurrentLine
	}
}

// wordToCurrentLine decides how to place the word into the output.
func wordToCurrentLine(cs *cellScanner) cellScannerState {
	wordCells := cs.wordCells()
	wordWidth := cs.wordWidth()

	if cs.posX+wordWidth <= cs.width {
		// Place the word onto the current line.
		cs.posX += wordWidth
		cs.line = append(cs.line, wordCells...)
		return scanCellRunes
	}
	return wrapWord
}

// wrapWord wraps the word onto the next line or lines.
func wrapWord(cs *cellScanner) cellScannerState {
	// Edge-case - the word starts the line and immediately doesn't fit.
	if cs.posX > 0 {
		cs.lines = append(cs.lines, cs.line)
		cs.posX = 0
		cs.line = nil
	}

	for i, wc := range cs.wordCells() {
		if i == 0 && wc.Rune == ' ' {
			// Skip the leading space when word wrapping.
			continue
		}

		if !runeWrapNeeded(wc.Rune, cs.posX, cs.width) {
			cs.posX += runewidth.RuneWidth(wc.Rune)
			cs.line = append(cs.line, wc)
			continue
		}

		// Replace the last placed rune with a dash indicating we wrapped the
		// word. Only do this for half-width runes.
		lastIdx := len(cs.line) - 1
		last := cs.line[lastIdx]
		lastRW := runewidth.RuneWidth(last.Rune)
		if cs.width > 1 && lastRW == 1 {
			cs.line[lastIdx] = buffer.NewCell('-', last.Opts)
			// Reset the scanner's position back to start scanning at the first
			// rune of this word that wasn't placed.
			cs.nextIdx = cs.wordStartIdx + i - 1
		} else {
			// Edge-case width is one, no space to put the dash rune.
			cs.nextIdx = cs.wordStartIdx + i
		}
		cs.atRunesInWord = true
		return scanCellRunes
	}

	cs.nextIdx = cs.wordEndIdx
	return scanCellRunes
}

// isWordCell determines if the cell contains a rune that belongs to a word.
func isWordCell(c *buffer.Cell) bool {
	if c == nil {
		return false
	}
	switch r := c.Rune; {
	case r == '\n':
	case r == ' ':
	default:
		return true
	}
	return false
}

// runeWrapNeeded returns true if wrapping is needed for the rune at the horizontal
// position on the canvas that has the specified width.
func runeWrapNeeded(r rune, posX, width int) bool {
	rw := runewidth.RuneWidth(r)
	return posX > width-rw
}
