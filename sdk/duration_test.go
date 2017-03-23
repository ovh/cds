package sdk

import (
	"testing"
	"time"
)

func TestRound(t *testing.T) {
	type args struct {
		d string
		r time.Duration
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1h23m45.6789s",
			args: args{d: "1h23m45.6789s", r: time.Second},
			want: "1h23m46s",
		},
		{
			name: "12.345678ms",
			args: args{d: "12.345678ms", r: time.Millisecond},
			want: "12ms",
		},
		{
			name: "1235h32m13.89s",
			args: args{d: "1235h32m13.89s", r: time.Minute},
			want: "1235h32m0s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, _ := time.ParseDuration(tt.args.d)
			if got := Round(d, tt.args.r).String(); got != tt.want {
				t.Errorf("Round() = %v, want %v", got, tt.want)
			}
		})
	}
}
