package amock

import (
	"fmt"
	"testing"

	"github.com/loopfz/gadgeto/amock/foo"
)

func TestMock(t *testing.T) {

	mock := NewMock()
	mock.Expect(foo.GetFoo, 200, foo.Foo{Identifier: "f1234", BarCount: 42}).OnIdentifier("f1234")
	foo.Client.Transport = mock

	fmt.Println("Step 1: get a foo with an identifier not matching the expected one")

	f, err := foo.GetFoo("f1")
	if err == nil {
		t.Error("Should not have returned foo object with non-matching ident")
	}

	fmt.Println("Step 1 returned:", err)

	fmt.Println("----------------------------------------------------------------------")

	fmt.Println("Step 2: get a foo with the correct identifier")

	f, err = foo.GetFoo("f1234")
	if err != nil {
		t.Error(err)
	}

	fmt.Println("Step 2 returned:", f)

	fmt.Println("----------------------------------------------------------------------")

	fmt.Println("Step 3: make the mock simulate a 503, get a foo expecting an error")

	mock.Expect(foo.GetFoo, 503, Raw([]byte(`<html><body><h1>503 Service Unavailable</h1>
No server is available to handle this request.
</body></html>`)))

	f, err = foo.GetFoo("f2")
	if err == nil {
		t.Error(err)
	}

	fmt.Println("Step 3 returned:", err)

	mock.AssertEmpty(t)
}
