package main

import (
	"fmt"
	tm "github.com/buger/goterm"
)

func main() {
	tm.Clear() // Clear current screen
	started := 100
	finished := 250

	// Based on http://golang.org/pkg/text/tabwriter
	totals := tm.NewTable(0, 10, 5, ' ', 0)
	fmt.Fprintf(totals, "Time\tStarted\tActive\tFinished\n")
	fmt.Fprintf(totals, "%s\t%d\t%d\t%d\n", "All", started, started-finished, finished)
	tm.Println(totals)

	tm.Flush()
}
