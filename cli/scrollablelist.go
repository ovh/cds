package cli

/*
Code from https://raw.githubusercontent.com/mikepea/go-jira-ui/a76c2a5cbfc0f3ccf063390e10957d3820af5d50/scrollablelist.go
Apache License
Version 2.0, January 2004
http://www.apache.org/licenses/
licence: https://github.com/mikepea/go-jira-ui/blob/master/LICENSE
*/

import (
	ui "github.com/gizak/termui"
)

// Default color values.
const (
	DefaultItemFgColor   = ui.ColorWhite
	DefaultItemBgColor   = ui.ColorBlack
	DefaultCursorFgColor = ui.ColorBlack
	DefaultCursorBgColor = ui.ColorWhite
)

// NewScrollableList returns a new *ScrollableList with current theme.
func NewScrollableList() *ScrollableList {
	return &ScrollableList{
		Block:         *ui.NewBlock(),
		ItemFgColor:   DefaultItemFgColor,
		ItemBgColor:   DefaultItemBgColor,
		CursorFgColor: DefaultCursorFgColor,
		CursorBgColor: DefaultCursorBgColor,
	}
}

// ScrollableList is a scrollable list with a cursor. To "deactivate" the cursor, just make the
// cursor colors the same as the item colors.
type ScrollableList struct {
	ui.Block

	header string

	// The items in the list
	items []string

	// The window's offset relative to the start of `items`
	offset int

	// The foreground color for non-cursor items
	ItemFgColor ui.Attribute

	// The background color for non-cursor items
	ItemBgColor ui.Attribute

	// The foreground color for the cursor
	CursorFgColor ui.Attribute

	// The background color for the cursor
	CursorBgColor ui.Attribute

	// The position of the cursor relative to the start of `items`
	cursor int

	cursorVisible bool
}

// SetCursorVisibility state.
func (sl *ScrollableList) SetCursorVisibility(b bool) { sl.cursorVisible = b }

// GetCursor returns cursor value.
func (sl *ScrollableList) GetCursor() int { return sl.cursor }

// GetItems returns list's items.
func (sl *ScrollableList) GetItems() []string { return sl.items }

// SetItems update list's items and fix cursor.
func (sl *ScrollableList) SetItems(is ...string) {
	sl.items = is

	h := sl.getInnerheight()

	// fix cursor and offset
	if len(sl.items) < h+sl.offset {
		sl.offset = 0
	}
	if len(sl.items) <= sl.cursor {
		sl.cursor = 0
	}
}

// SetHeader to list.
func (sl *ScrollableList) SetHeader(h string) { sl.header = h }

func (sl *ScrollableList) render() { ui.Render(sl) }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Buffer implements the termui.Bufferer interface
func (sl *ScrollableList) Buffer() ui.Buffer {
	b := sl.Block.Buffer()

	h := sl.getInnerheight()
	startItem, endItem := sl.offset, min(sl.offset+h, len(sl.items))

	var idx int
	if sl.header != "" {
		sl.printLine(b, sl.header, idx, DefaultItemFgColor, DefaultItemBgColor)
		idx++
	}

	for i, item := range sl.items[startItem:endItem] {
		fg, bg := sl.ItemFgColor, sl.ItemBgColor
		if i+startItem == sl.cursor && sl.cursorVisible {
			fg, bg = sl.CursorFgColor, sl.CursorBgColor
		}
		sl.printLine(b, item, idx, fg, bg)
		idx++
	}

	return b
}

func (sl *ScrollableList) printLine(b ui.Buffer, l string, index int, fg, bg ui.Attribute) {
	if l == "" {
		l = " "
	}

	cells := ui.DefaultTxBuilder.Build(l, fg, bg)
	cells = ui.DTrimTxCls(cells, sl.InnerWidth())
	offsetX := 0
	for _, cell := range cells {
		width := cell.Width()
		b.Set(
			sl.InnerBounds().Min.X+offsetX,
			sl.InnerBounds().Min.Y+index,
			cell,
		)
		offsetX += width
	}
}

func (sl *ScrollableList) getInnerheight() int {
	h := sl.InnerHeight()
	if sl.header != "" {
		return h - 1
	}
	return h
}

// CursorDown move the cursor down one row; moving the cursor out of the window will cause
// scrolling.
func (sl *ScrollableList) CursorDown() {
	if sl.cursor < len(sl.items)-1 {
		sl.cursor++
	}

	h := sl.getInnerheight()
	if sl.cursor >= h+sl.offset {
		sl.offset = (sl.cursor - h) + 1
	}

	sl.render()
}

// CursorUp move the cursor up one row; moving the cursor out of the window will cause
// scrolling.
func (sl *ScrollableList) CursorUp() {
	if sl.cursor > 0 {
		sl.cursor--
	}
	if sl.cursor < sl.offset {
		sl.offset = sl.cursor
	}

	sl.render()
}
