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

// Package terminalapi defines the API of all terminal implementations.
package terminalapi

import (
	"context"
	"image"

	"github.com/mum4k/termdash/cell"
)

// Terminal abstracts an implementation of a 2-D terminal.
// A terminal consists of a number of cells.
type Terminal interface {
	// Size returns the terminal width and height in cells.
	Size() image.Point

	// Clear clears the content of the internal back buffer, resetting all
	// cells to their default content and attributes. Sets the provided options
	// on all the cell.
	Clear(opts ...cell.Option) error
	// Flush flushes the internal back buffer to the terminal.
	Flush() error

	// SetCursor sets the position of the cursor.
	SetCursor(p image.Point)
	// HideCursos hides the cursor.
	HideCursor()

	// SetCell sets the value of the specified cell to the provided rune.
	// Use the options to specify which attributes to modify, if an attribute
	// option isn't specified, the attribute retains its previous value.
	SetCell(p image.Point, r rune, opts ...cell.Option) error

	// Event waits for the next event and returns it.
	// This call blocks until the next event or cancellation of the context.
	// Returns nil when the context gets canceled.
	Event(ctx context.Context) Event
}
