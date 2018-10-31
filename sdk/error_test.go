package sdk

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
			"Check for wrapped Error is true",
			args{
				err: WithStack(ErrNoProject),
				t:   ErrNoProject,
			},
			true,
		},
		{
			"Check for Error is true",
			args{
				err: ErrNoProject,
				t:   ErrNoProject,
			},
			true,
		},
		{
			"Check for other error is false",
			args{
				err: fmt.Errorf("project does not exist"),
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
	assert.Equal(t, "TestNewError: this is an error generated from vendor (caused by: wrong request)", err.Error())

	// print the error call stack
	t.Log(err)
	// print the error stack trace
	t.Logf("%+v\n", err)

	httpErr := ExtractHTTPError(err, "fr")

	// print the http error
	t.Log(httpErr)

	assert.Equal(t, "this is an error generated from vendor (caused by: wrong request)", httpErr.Error())
}

func TestWrapError(t *testing.T) {
	err := oneForStackTest()
	assert.Equal(t, "TestWrapError>oneForStackTest>twoForStackTest>threeForStackTest>fourForStackTest>fiveForStackTest: internal server error (caused by: one: two: three: four: five: this is an error generated from vendor)", err.Error())

	// print the error call stack
	t.Log(err)
	// print the error stack trace
	t.Logf("%+v\n", err)
}

func oneForStackTest() error   { return WrapError(twoForStackTest(), "one") }
func twoForStackTest() error   { return WrapError(threeForStackTest(), "two") }
func threeForStackTest() error { return WrapError(fourForStackTest(), "three") }
func fourForStackTest() error  { return WrapError(fiveForStackTest(), "four") }
func fiveForStackTest() error {
	return WrapError(fmt.Errorf("this is an error generated from vendor"), "five")
}

func TestCause(t *testing.T) {
	err := oneForStackTest()
	cause := Cause(err)
	assert.Equal(t, "this is an error generated from vendor", cause.Error())

	err = NewError(ErrActionLoop, WrapError(WithStack(sql.ErrNoRows), "more info"))
	cause = Cause(err)
	assert.Equal(t, sql.ErrNoRows, cause)
	assert.NotEqual(t, sql.ErrNoRows, err)

	err = sql.ErrConnDone
	cause = Cause(err)
	assert.Equal(t, sql.ErrConnDone, cause)
}
