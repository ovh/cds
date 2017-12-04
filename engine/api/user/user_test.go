package user

import "testing"

func TestIsAllowedDomain(t *testing.T) {
	type args struct {
		allowedDomains string
		email          string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "all",
			args: args{
				allowedDomains: "",
				email:          "user@a-domain.com",
			},
			want: true,
		},
		{
			name: "valid domain",
			args: args{
				allowedDomains: "a-domain.com",
				email:          "user@a-domain.com",
			},
			want: true,
		},
		{
			name: "not valid domain",
			args: args{
				allowedDomains: "another-domain.com",
				email:          "user@a-domain.com",
			},
			want: false,
		},
		{
			name: "not valid domain, with two '@'",
			args: args{
				allowedDomains: "a-domain.com",
				email:          "user@foo.com@a-domain.com",
			},
			want: false,
		},
		{
			name: "two domains, try first",
			args: args{
				allowedDomains: "aa.com,bb.com",
				email:          "user@aa.com",
			},
			want: true,
		},
		{
			name: "two domains, try second",
			args: args{
				allowedDomains: "aa.com,bb.com",
				email:          "user@bb.com",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAllowedDomain(tt.args.allowedDomains, tt.args.email); got != tt.want {
				t.Errorf("IsAllowedDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}
