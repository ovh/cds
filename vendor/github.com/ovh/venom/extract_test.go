package venom

import "testing"

func Test_RemoveNotPrintableChar(t *testing.T) {
	type args struct {
		in string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "remove U+001B espace code (not printable)",
			// line below contains escapce code U+001B
			args: args{in: "python-mysqldb : [34mOK[0m"},
			want: "python-mysqldb :  [34mOK [0m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveNotPrintableChar(tt.args.in); got != tt.want {
				t.Errorf("RemoveNotPrintableChar() = %v, want %v", got, tt.want)
			}
		})
	}
}
