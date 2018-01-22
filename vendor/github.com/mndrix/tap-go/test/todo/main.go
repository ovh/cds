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

	t.Header(6)
	t.TODO = true
	t.Ok(false, "using Ok(false, ...) in TODO mode")
	t.Fail("using Fail(...) in TODO mode")
	t.TODO = false
	t.Ok(true, "using Ok(false, ...) after leaving TODO mode")

	t.Todo().Fail("using Fail(...) in TODO mode with method chaining")
	t.Pass("using Pass(...) after Todo method chaining")

	got := buf.String()
	t.Ok(got == expected, "TODO gave expected output")
}

const expected = `TAP version 13
1..6
not ok 1 # TODO using Ok(false, ...) in TODO mode
not ok 2 # TODO using Fail(...) in TODO mode
ok 3 - using Ok(false, ...) after leaving TODO mode
not ok 4 # TODO using Fail(...) in TODO mode with method chaining
ok 5 - using Pass(...) after Todo method chaining
`
