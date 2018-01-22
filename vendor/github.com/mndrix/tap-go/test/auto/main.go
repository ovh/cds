package main

import "github.com/mndrix/tap-go"

func main() {
	t := tap.New()
	t.Header(0)
	t.Ok(true, "first test")
	t.Ok(true, "second test")
	t.AutoPlan()
}
