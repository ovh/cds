package main

import (
	"bytes"

	"github.com/mndrix/tap-go"
)

func main() {
	t1 := tap.New()
	t1.Header(2)

	buf := new(bytes.Buffer)
	t2 := tap.New()
	t2.Writer = buf
	t2.Header(2)

	buf.Reset()
	t2.Ok(false, "first test")
	t1.Ok(buf.String() == "not ok 1 - first test\n", "Ok(false, ...) produces appropriate output")

	buf.Reset()
	t2.Fail("second test")
	t1.Ok(buf.String() == "not ok 2 - second test\n", "Fail(...) produces appropriate output")
}
