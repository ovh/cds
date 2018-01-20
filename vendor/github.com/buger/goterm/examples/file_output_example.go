package main

import (
	"bufio"
	"fmt"
	tm "github.com/buger/goterm"
	"os"
)

func main() {
	f, err := os.Create("box.txt")
	if err != nil {
		panic("Unable to create box file!")
	}
	defer f.Close()

	// Tell tm to use the file we just opened, not stdout
	tm.Output = bufio.NewWriter(f)

	// More or less stolen from the box example
	tm.Clear()
	box := tm.NewBox(30|tm.PCT, 20, 0)
	fmt.Fprint(box, "Some box content")
	tm.Print(tm.MoveTo(box.String(), 40|tm.PCT, 40|tm.PCT))
	tm.Flush()

	fmt.Println("Now view the contents of 'box.txt' in an ansi-capable terminal")
}
