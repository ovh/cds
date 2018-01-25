package main

import (
	"bytes"
	"io"
	"os"

	tap "github.com/mndrix/tap-go"
)

func main() {
	// collect output for comparison later
	buf := new(bytes.Buffer)
	t := tap.New()
	t.Writer = io.MultiWriter(os.Stdout, buf)

	t.Header(4)
	t.Skip(1, "insufficient flogiston pressure")
	t.Skip(2, "no /sys directory")

	got := buf.String()
	t.Ok(got == expected, "skip gave expected output")
}

const expected = `TAP version 13
1..4
ok 1 # SKIP insufficient flogiston pressure
ok 2 # SKIP no /sys directory
ok 3 # SKIP no /sys directory
`
