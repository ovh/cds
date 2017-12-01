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
