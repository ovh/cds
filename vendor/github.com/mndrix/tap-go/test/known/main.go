package main

import "github.com/mndrix/tap-go"

func main() {
	t := tap.New()
	t.Header(2)
	t.Ok(true, "first test")
	t.Pass("second test")
}
