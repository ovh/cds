package main

import (
	tm "github.com/buger/goterm"
	"time"
)

func main() {
	tm.Clear() // Clear current screen

	for {
		// By moving cursor to top-left position we ensure that console output
		// will be overwritten each time, instead of adding new.
		tm.MoveCursor(1, 1)

		tm.Println("Current Time:", time.Now().Format(time.RFC1123))

		tm.Flush() // Call it every time at the end of rendering

		time.Sleep(time.Second)
	}
}
