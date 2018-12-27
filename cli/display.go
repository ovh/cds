package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	surterm "gopkg.in/AlecAivazis/survey.v1/terminal"

	"golang.org/x/crypto/ssh/terminal"
)

// Display helps you to display message on a terminal
type Display string

// Printf update the displayed message
func (d *Display) Printf(format string, args ...interface{}) {
	*d = Display(fmt.Sprintf(format, args...))
}

// Do runs a goroutine which update the display
func (d *Display) Do(ctx context.Context) {
	clear := "\r"
	w, _, _ := terminal.GetSize(1)
	for i := 0; i < w; i++ {
		clear += " "
	}

	cursor := surterm.Cursor{Out: os.Stdout}

	var count int
	go func(d *Display) {
		for {
			time.Sleep(100 * time.Millisecond)
			if *d == "" || ctx.Err() != nil {
				continue
			}

			for i := 0; i < count-1; i++ {
				fmt.Printf(clear)
				cursor.PreviousLine(1)
			}
			count = len(strings.Split(string(*d), "\n"))

			fmt.Printf(clear + "\r" + string(*d))
			*d = ""
		}
	}(d)
}
