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
	err := fmt.Errorf("this is an error generated from vendor")
	cdsErr := NewError(ErrWrongRequest, err)
	// print the CDS error line
	fmt.Printf("%s\n", cdsErr)
	// print the root error message and stack trace
	fmt.Printf("%+v\n", cdsErr.(Error).Root)
}

func TestWrapError(t *testing.T) {
	err := oneForStackTest()
	// print the CDS error line
	fmt.Printf("%s\n", err)
	// print the root error message and stack trace
	fmt.Printf("%+v\n", err.(Error).Root)
}

func oneForStackTest() error   { return twoForStackTest() }
func twoForStackTest() error   { return threeForStackTest() }
func threeForStackTest() error { return fourForStackTest() }
func fourForStackTest() error  { return fiveForStackTest() }
func fiveForStackTest() error {
	err := fmt.Errorf("this is an error generated from vendor")
	return WrapError(err, "cds custom message %d", 50)
}
