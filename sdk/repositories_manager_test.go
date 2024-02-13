package sdk

import "testing"

func Test_isSameCommit(t *testing.T) {
	type args struct {
		sha1  string
		sha1b string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "same",
			args: args{sha1: "aaaaaa", sha1b: "aaaaaa"},
			want: true,
		},
		{
			name: "same",
			args: args{sha1: "4e269fccb82a", sha1b: "4e269fccb82a1b98a510b172b2c8db8ec9b4abb0"},
			want: true,
		},
		{
			name: "same",
			args: args{sha1: "4e269fccb82a1b98a510b172b2c8db8ec9b4abb0", sha1b: "4e269fccb82a"},
			want: true,
		},
		{
			name: "not same",
			args: args{sha1: "aa4e269fccb82a1b98a510b172b2c8db8ec9b4abb0", sha1b: "aa4e269fccb82a"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VCSIsSameCommit(tt.args.sha1, tt.args.sha1b); got != tt.want {
				t.Errorf("isSameCommit() = %v, want %v", got, tt.want)
			}
		})
	}
}
