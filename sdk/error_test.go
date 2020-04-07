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
			"Check for NewErrorFrom is true",
			args{
				err: NewErrorFrom(ErrNoProject, "My from value"),
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
	assert.Equal(t, "TestNewError: wrong request (from: this is an error generated from vendor)", err.Error())

	// print the error call stack
	t.Log(err)
	// print the error stack trace
	t.Logf("%+v\n", err)

	httpErr := ExtractHTTPError(err, "fr")

	// print the http error
	t.Log(httpErr)

	assert.Equal(t, "la requête est incorrecte (from: this is an error generated from vendor)", httpErr.Error())
}

func TestWrapError(t *testing.T) {
	err := fourForStackTest()
	assert.Equal(t, "TestWrapError>fourForStackTest>fiveForStackTest: internal server error (caused by: four: five: this is an error generated from vendor)", err.Error())

	err = oneForStackTest()
	assert.Equal(t, "TestWrapError>oneForStackTest>twoForStackTest>threeForStackTest>fourForStackTest>fiveForStackTest: action definition contains a recursive loop (caused by: one: two: three: four: five: this is an error generated from vendor)", err.Error())

	// print the error call stack
	t.Log(err)
	// print the error stack trace
	t.Logf("%+v\n", err)
}

func oneForStackTest() error   { return WrapError(twoForStackTest(), "one") }
func twoForStackTest() error   { return WrapError(threeForStackTest(), "two") }
func threeForStackTest() error { return NewError(ErrActionLoop, WrapError(fourForStackTest(), "three")) }
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

func TestNewAdvancedError(t *testing.T) {
	err := NewErrorFrom(ErrWrongRequest, "this is an error generated from vendor")
	assert.Equal(t, "TestNewAdvancedError>NewErrorFrom: wrong request (from: this is an error generated from vendor)", err.Error())
	httpErr := ExtractHTTPError(err, "fr")
	assert.Equal(t, "la requête est incorrecte (from: this is an error generated from vendor)", httpErr.Error())

	err = WrapError(err, "Something no visible for http error")
	assert.Equal(t, "TestNewAdvancedError>NewErrorFrom: wrong request (from: this is an error generated from vendor) (caused by: Something no visible for http error: this is an error generated from vendor)", err.Error())
	httpErr = ExtractHTTPError(err, "fr")
	assert.Equal(t, "la requête est incorrecte (from: this is an error generated from vendor)", httpErr.Error())

	err = NewError(ErrAlreadyTaken, err)
	assert.Equal(t, "TestNewAdvancedError>NewErrorFrom: This job is already taken by another worker (from: this is an error generated from vendor) (caused by: Something no visible for http error: this is an error generated from vendor)", err.Error())
	httpErr = ExtractHTTPError(err, "fr")
	assert.Equal(t, "Ce job est déjà en cours de traitement par un autre worker (from: this is an error generated from vendor)", httpErr.Error())

	err = NewErrorWithStack(err, NewErrorFrom(ErrNotFound, "can't found this"))
	assert.Equal(t, "TestNewAdvancedError>NewErrorFrom: resource not found (from: can't found this) (caused by: Something no visible for http error: this is an error generated from vendor)", err.Error())
	httpErr = ExtractHTTPError(err, "fr")
	assert.Equal(t, "la ressource n'existe pas (from: can't found this)", httpErr.Error())
}
