package sdk

import (
	"fmt"
	"testing"
)

func TestErrorIs(t *testing.T) {
	type args struct {
		err error
		t   Error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Check Error is true",
			args{
				err: fmt.Errorf(ErrNoProject.String()),
				t:   ErrNoProject,
			},
			true,
		},
		{
			"Check Error is false",
			args{
				err: fmt.Errorf("FOO"),
				t:   ErrNoProject,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorIs(tt.args.err, tt.args.t); got != tt.want {
				t.Errorf("ErrorIs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrWrongRequest, fmt.Errorf("this is an error generated from vendor"))
	// print the error call stack
	fmt.Println(err)
	// print the error stack trace
	fmt.Printf("%+v\n", err)

	httpErr := ExtractHTTPError(err, "fr")
	// print the http error
	fmt.Println(httpErr)
}

func TestWrapError(t *testing.T) {
	err := oneForStackTest()
	// print the error call stack
	fmt.Println(err)
	// print the error stack trace
	fmt.Printf("%+v\n", err)
}

func oneForStackTest() error   { return WrapError(twoForStackTest(), "one") }
func twoForStackTest() error   { return WrapError(threeForStackTest(), "two") }
func threeForStackTest() error { return WrapError(fourForStackTest(), "three") }
func fourForStackTest() error  { return WrapError(fiveForStackTest(), "four") }
func fiveForStackTest() error {
	return WrapError(fmt.Errorf("this is an error generated from vendor"), "five")
}
