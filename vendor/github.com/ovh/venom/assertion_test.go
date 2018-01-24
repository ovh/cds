package venom

import (
	"reflect"
	"testing"
)

func Test_splitAssertion(t *testing.T) {
	for _, tt := range []struct {
		Assertion string
		Args      []string
	}{
		{Assertion: `cmd arg`, Args: []string{"cmd", "arg"}},
		{Assertion: `cmd arg1 "arg 2"`, Args: []string{"cmd", "arg1", "arg 2"}},
		{Assertion: `cmd 'arg 1' "arg 2"`, Args: []string{"cmd", "arg 1", "arg 2"}},
		{Assertion: `cmd 'arg 1' "'arg' 2"`, Args: []string{"cmd", "arg 1", "'arg' 2"}},
		{Assertion: `cmd '"arg 1"' "'arg' 2"`, Args: []string{"cmd", "\"arg 1\"", "'arg' 2"}},
	} {
		args := splitAssertion(tt.Assertion)
		if !reflect.DeepEqual(args, tt.Args) {
			t.Errorf("expected args to be equal to %#v, got %#v", tt.Args, args)
		}
	}
}
