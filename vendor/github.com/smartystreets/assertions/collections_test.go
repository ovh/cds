package assertions

import (
	"fmt"
	"testing"
	"time"
)

func TestShouldContainKey(t *testing.T) {
	fail(t, so(map[int]int{}, ShouldContainKey), "This assertion requires exactly 1 comparison values (you provided 0).")
	fail(t, so(map[int]int{}, ShouldContainKey, 1, 2, 3), "This assertion requires exactly 1 comparison values (you provided 3).")

	fail(t, so(Thing1{}, ShouldContainKey, 1), "You must provide a valid map type (was assertions.Thing1)!")
	fail(t, so(nil, ShouldContainKey, 1), "You must provide a valid map type (was <nil>)!")
	fail(t, so(map[int]int{1: 41}, ShouldContainKey, 2), "Expected the map[int]int to contain the key: [2] (but it didn't)!")

	pass(t, so(map[int]int{1: 41}, ShouldContainKey, 1))
	pass(t, so(map[int]int{1: 41, 2: 42, 3: 43}, ShouldContainKey, 2))
}

func TestShouldNotContainKey(t *testing.T) {
	fail(t, so(map[int]int{}, ShouldNotContainKey), "This assertion requires exactly 1 comparison values (you provided 0).")
	fail(t, so(map[int]int{}, ShouldNotContainKey, 1, 2, 3), "This assertion requires exactly 1 comparison values (you provided 3).")

	fail(t, so(Thing1{}, ShouldNotContainKey, 1), "You must provide a valid map type (was assertions.Thing1)!")
	fail(t, so(nil, ShouldNotContainKey, 1), "You must provide a valid map type (was <nil>)!")
	fail(t, so(map[int]int{1: 41}, ShouldNotContainKey, 1), "Expected the map[int]int NOT to contain the key: [1] (but it did)!")
	pass(t, so(map[int]int{1: 41}, ShouldNotContainKey, 2))
}

func TestShouldContain(t *testing.T) {
	fail(t, so([]int{}, ShouldContain), "This assertion requires exactly 1 comparison values (you provided 0).")
	fail(t, so([]int{}, ShouldContain, 1, 2, 3), "This assertion requires exactly 1 comparison values (you provided 3).")

	fail(t, so(Thing1{}, ShouldContain, 1), "You must provide a valid container (was assertions.Thing1)!")
	fail(t, so(nil, ShouldContain, 1), "You must provide a valid container (was <nil>)!")
	fail(t, so([]int{1}, ShouldContain, 2), "Expected the container ([]int) to contain: '2' (but it didn't)!")
	fail(t, so([][]int{[]int{1}}, ShouldContain, []int{2}), "Expected the container ([][]int) to contain: '[2]' (but it didn't)!")

	pass(t, so([]int{1}, ShouldContain, 1))
	pass(t, so([]int{1, 2, 3}, ShouldContain, 2))
	pass(t, so([][]int{[]int{1}, []int{2}, []int{3}}, ShouldContain, []int{2}))
}

func TestShouldNotContain(t *testing.T) {
	fail(t, so([]int{}, ShouldNotContain), "This assertion requires exactly 1 comparison values (you provided 0).")
	fail(t, so([]int{}, ShouldNotContain, 1, 2, 3), "This assertion requires exactly 1 comparison values (you provided 3).")

	fail(t, so(Thing1{}, ShouldNotContain, 1), "You must provide a valid container (was assertions.Thing1)!")
	fail(t, so(nil, ShouldNotContain, 1), "You must provide a valid container (was <nil>)!")

	fail(t, so([]int{1}, ShouldNotContain, 1), "Expected the container ([]int) NOT to contain: '1' (but it did)!")
	fail(t, so([]int{1, 2, 3}, ShouldNotContain, 2), "Expected the container ([]int) NOT to contain: '2' (but it did)!")
	fail(t, so([][]int{[]int{1}, []int{2}, []int{3}}, ShouldNotContain, []int{2}), "Expected the container ([][]int) NOT to contain: '[2]' (but it did)!")

	pass(t, so([]int{1}, ShouldNotContain, 2))
	pass(t, so([][]int{[]int{1}, []int{2}, []int{3}}, ShouldNotContain, []int{4}))
}

func TestShouldBeIn(t *testing.T) {
	fail(t, so(4, ShouldBeIn), needNonEmptyCollection)

	container := []int{1, 2, 3, 4}
	pass(t, so(4, ShouldBeIn, container))
	pass(t, so(4, ShouldBeIn, 1, 2, 3, 4))
	pass(t, so([]int{4}, ShouldBeIn, [][]int{[]int{1}, []int{2}, []int{3}, []int{4}}))
	pass(t, so([]int{4}, ShouldBeIn, []int{1}, []int{2}, []int{3}, []int{4}))

	fail(t, so(4, ShouldBeIn, 1, 2, 3), "Expected '4' to be in the container ([]interface {}), but it wasn't!")
	fail(t, so(4, ShouldBeIn, []int{1, 2, 3}), "Expected '4' to be in the container ([]int), but it wasn't!")
	fail(t, so([]int{4}, ShouldBeIn, []int{1}, []int{2}, []int{3}), "Expected '[4]' to be in the container ([]interface {}), but it wasn't!")
	fail(t, so([]int{4}, ShouldBeIn, [][]int{[]int{1}, []int{2}, []int{3}}), "Expected '[4]' to be in the container ([][]int), but it wasn't!")
}

func TestShouldNotBeIn(t *testing.T) {
	fail(t, so(4, ShouldNotBeIn), needNonEmptyCollection)

	container := []int{1, 2, 3, 4}
	pass(t, so(42, ShouldNotBeIn, container))
	pass(t, so(42, ShouldNotBeIn, 1, 2, 3, 4))
	pass(t, so([]int{42}, ShouldNotBeIn, []int{1}, []int{2}, []int{3}, []int{4}))
	pass(t, so([]int{42}, ShouldNotBeIn, [][]int{[]int{1}, []int{2}, []int{3}, []int{4}}))

	fail(t, so(2, ShouldNotBeIn, 1, 2, 3), "Expected '2' NOT to be in the container ([]interface {}), but it was!")
	fail(t, so(2, ShouldNotBeIn, []int{1, 2, 3}), "Expected '2' NOT to be in the container ([]int), but it was!")
	fail(t, so([]int{2}, ShouldNotBeIn, []int{1}, []int{2}, []int{3}), "Expected '[2]' NOT to be in the container ([]interface {}), but it was!")
	fail(t, so([]int{2}, ShouldNotBeIn, [][]int{[]int{1}, []int{2}, []int{3}}), "Expected '[2]' NOT to be in the container ([][]int), but it was!")
}

func TestShouldBeEmpty(t *testing.T) {
	fail(t, so(1, ShouldBeEmpty, 2, 3), "This assertion requires exactly 0 comparison values (you provided 2).")

	pass(t, so([]int{}, ShouldBeEmpty))           // empty slice
	pass(t, so([][]int{}, ShouldBeEmpty))         // empty slice
	pass(t, so([]interface{}{}, ShouldBeEmpty))   // empty slice
	pass(t, so(map[string]int{}, ShouldBeEmpty))  // empty map
	pass(t, so("", ShouldBeEmpty))                // empty string
	pass(t, so(&[]int{}, ShouldBeEmpty))          // pointer to empty slice
	pass(t, so(&[0]int{}, ShouldBeEmpty))         // pointer to empty array
	pass(t, so(nil, ShouldBeEmpty))               // nil
	pass(t, so(make(chan string), ShouldBeEmpty)) // empty channel

	fail(t, so([]int{1}, ShouldBeEmpty), "Expected [1] to be empty (but it wasn't)!")                      // non-empty slice
	fail(t, so([][]int{[]int{1}}, ShouldBeEmpty), "Expected [[1]] to be empty (but it wasn't)!")           // non-empty slice
	fail(t, so([]interface{}{1}, ShouldBeEmpty), "Expected [1] to be empty (but it wasn't)!")              // non-empty slice
	fail(t, so(map[string]int{"hi": 0}, ShouldBeEmpty), "Expected map[hi:0] to be empty (but it wasn't)!") // non-empty map
	fail(t, so("hi", ShouldBeEmpty), "Expected hi to be empty (but it wasn't)!")                           // non-empty string
	fail(t, so(&[]int{1}, ShouldBeEmpty), "Expected &[1] to be empty (but it wasn't)!")                    // pointer to non-empty slice
	fail(t, so(&[1]int{1}, ShouldBeEmpty), "Expected &[1] to be empty (but it wasn't)!")                   // pointer to non-empty array
	c := make(chan int, 1)                                                                                 // non-empty channel
	go func() { c <- 1 }()
	time.Sleep(time.Millisecond)
	fail(t, so(c, ShouldBeEmpty), fmt.Sprintf("Expected %+v to be empty (but it wasn't)!", c))
}

func TestShouldNotBeEmpty(t *testing.T) {
	fail(t, so(1, ShouldNotBeEmpty, 2, 3), "This assertion requires exactly 0 comparison values (you provided 2).")

	fail(t, so([]int{}, ShouldNotBeEmpty), "Expected [] to NOT be empty (but it was)!")             // empty slice
	fail(t, so([]interface{}{}, ShouldNotBeEmpty), "Expected [] to NOT be empty (but it was)!")     // empty slice
	fail(t, so(map[string]int{}, ShouldNotBeEmpty), "Expected map[] to NOT be empty (but it was)!") // empty map
	fail(t, so("", ShouldNotBeEmpty), "Expected  to NOT be empty (but it was)!")                    // empty string
	fail(t, so(&[]int{}, ShouldNotBeEmpty), "Expected &[] to NOT be empty (but it was)!")           // pointer to empty slice
	fail(t, so(&[0]int{}, ShouldNotBeEmpty), "Expected &[] to NOT be empty (but it was)!")          // pointer to empty array
	fail(t, so(nil, ShouldNotBeEmpty), "Expected <nil> to NOT be empty (but it was)!")              // nil
	c := make(chan int, 0)                                                                          // non-empty channel
	fail(t, so(c, ShouldNotBeEmpty), fmt.Sprintf("Expected %+v to NOT be empty (but it was)!", c))  // empty channel

	pass(t, so([]int{1}, ShouldNotBeEmpty))                // non-empty slice
	pass(t, so([]interface{}{1}, ShouldNotBeEmpty))        // non-empty slice
	pass(t, so(map[string]int{"hi": 0}, ShouldNotBeEmpty)) // non-empty map
	pass(t, so("hi", ShouldNotBeEmpty))                    // non-empty string
	pass(t, so(&[]int{1}, ShouldNotBeEmpty))               // pointer to non-empty slice
	pass(t, so(&[1]int{1}, ShouldNotBeEmpty))              // pointer to non-empty array
	c = make(chan int, 1)
	go func() { c <- 1 }()
	time.Sleep(time.Millisecond)
	pass(t, so(c, ShouldNotBeEmpty))
}

func TestShouldHaveLength(t *testing.T) {
	fail(t, so(1, ShouldHaveLength, 2), "You must provide a valid container (was int)!")
	fail(t, so(nil, ShouldHaveLength, 1), "You must provide a valid container (was <nil>)!")
	fail(t, so("hi", ShouldHaveLength, float64(1.0)), "You must provide a valid integer (was float64)!")
	fail(t, so([]string{}, ShouldHaveLength), "This assertion requires exactly 1 comparison values (you provided 0).")
	fail(t, so([]string{}, ShouldHaveLength, 1, 2), "This assertion requires exactly 1 comparison values (you provided 2).")
	fail(t, so([]string{}, ShouldHaveLength, -10), "You must provide a valid positive integer (was -10)!")

	fail(t, so([]int{}, ShouldHaveLength, 1), "Expected [] (length: 0) to have length equal to '1', but it wasn't!")             // empty slice
	fail(t, so([]interface{}{}, ShouldHaveLength, 1), "Expected [] (length: 0) to have length equal to '1', but it wasn't!")     // empty slice
	fail(t, so(map[string]int{}, ShouldHaveLength, 1), "Expected map[] (length: 0) to have length equal to '1', but it wasn't!") // empty map
	fail(t, so("", ShouldHaveLength, 1), "Expected  (length: 0) to have length equal to '1', but it wasn't!")                    // empty string
	fail(t, so(&[]int{}, ShouldHaveLength, 1), "Expected &[] (length: 0) to have length equal to '1', but it wasn't!")           // pointer to empty slice
	fail(t, so(&[0]int{}, ShouldHaveLength, 1), "Expected &[] (length: 0) to have length equal to '1', but it wasn't!")          // pointer to empty array
	c := make(chan int, 0)                                                                                                       // non-empty channel
	fail(t, so(c, ShouldHaveLength, 1), fmt.Sprintf("Expected %+v (length: 0) to have length equal to '1', but it wasn't!", c))
	c = make(chan int) // empty channel
	fail(t, so(c, ShouldHaveLength, 1), fmt.Sprintf("Expected %+v (length: 0) to have length equal to '1', but it wasn't!", c))

	pass(t, so([]int{1}, ShouldHaveLength, 1))                // non-empty slice
	pass(t, so([]interface{}{1}, ShouldHaveLength, 1))        // non-empty slice
	pass(t, so(map[string]int{"hi": 0}, ShouldHaveLength, 1)) // non-empty map
	pass(t, so("hi", ShouldHaveLength, 2))                    // non-empty string
	pass(t, so(&[]int{1}, ShouldHaveLength, 1))               // pointer to non-empty slice
	pass(t, so(&[1]int{1}, ShouldHaveLength, 1))              // pointer to non-empty array
	c = make(chan int, 1)
	go func() { c <- 1 }()
	time.Sleep(time.Millisecond)
	pass(t, so(c, ShouldHaveLength, 1))

}
