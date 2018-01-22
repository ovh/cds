package main

import "github.com/mndrix/tap-go"

func main() {
	add := func(x int) bool { return x+3 > x }
	sub := func(x int) bool { return x-3 < x }
	one := func(x int) bool { return x*1 == x }

	t := tap.New()
	t.Header(0)
	t.Check(add, "addition makes numbers larger")
	t.Check(sub, "subtraction makes numbers smaller")
	t.Check(one, "1 is a multiplicative identity")
	t.AutoPlan()
}
