package main

import (
	"fmt"
	tm "github.com/buger/goterm"
)

func main() {
	tm.Clear()

	// Create Box with 30% width of current screen, and height of 20 lines
	box := tm.NewBox(30|tm.PCT, 20, 0)

	// Add some content to the box
	// Note that you can add ANY content, even tables
	fmt.Fprint(box, "Some box content")

	// Move Box to approx center of the screen
	tm.Print(tm.MoveTo(box.String(), 40|tm.PCT, 40|tm.PCT))

	tm.Flush()
}
