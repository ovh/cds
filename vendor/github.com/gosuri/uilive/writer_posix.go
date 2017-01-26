// +build !windows

package uilive

import (
	"fmt"
)

func (w *Writer) clearLines() {
	for i := 0; i < w.lineCount; i++ {
		fmt.Fprintf(w.Out, "%c[%dA", ESC, 0) // move the cursor up
		fmt.Fprintf(w.Out, "%c[2K\r", ESC)   // clear the line
	}
}
