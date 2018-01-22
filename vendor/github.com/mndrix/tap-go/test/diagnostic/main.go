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

	t.Header(1)
	t.Diagnostic("expecting all to be well")
	t.Diagnosticf("here's some perfectly magical output: %d %s 0x%X.", 6, "abracadabra", 28)
	t.Diagnostic("some\nmultiline\ntext\n")
	t.Diagnosticf("%d lines\n%s multiline\ntext", 3, "more")

	got := buf.String()
	t.Ok(got == expected, "diagnostics gave expected output")
}

const expected = `TAP version 13
1..1
# expecting all to be well
# here's some perfectly magical output: 6 abracadabra 0x1C.
# some
# multiline
# text
# 3 lines
# more multiline
# text
`
